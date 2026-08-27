package scraperequestintake_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/scraperequestintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const (
	scrapeRequestURL = "https://example.com/"
	pageTitle        = "Hi"
)

type fakeFetch struct {
	outcome pagefetch.FetchOutcome
	err     error
	mu      sync.Mutex
	urls    []string
}

func (f *fakeFetch) Fetch(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	f.mu.Lock()
	f.urls = append(f.urls, pageURL.String())
	f.mu.Unlock()

	return f.outcome, f.err
}

type recordingURLs struct {
	receipt  urlmeta.Receipt
	err      error
	received []yacymodel.URLMetadata
}

func (r *recordingURLs) Receive(
	_ context.Context,
	metadata []yacymodel.URLMetadata,
) (urlmeta.Receipt, error) {
	r.received = append(r.received, metadata...)

	return r.receipt, r.err
}

type recordingPostings struct {
	receipt rwiadmission.Receipt
	err     error
	calls   [][]yacymodel.RWIPosting
}

func (r *recordingPostings) Receive(
	_ context.Context,
	postings []yacymodel.RWIPosting,
) (rwiadmission.Receipt, error) {
	r.calls = append(r.calls, postings)

	return r.receipt, r.err
}

func scrapeRequestMessage(t *testing.T) *pullintaketest.Message {
	t.Helper()

	data, err := scraperequestcontract.MarshalScrapeRequest(
		scraperequestcontract.ScrapeRequest{
			PageURL: canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
		},
	)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return &pullintaketest.Message{Body: data}
}

func archivedScrapeRequestMessage(
	t *testing.T,
	fetchURL string,
) *pullintaketest.Message {
	t.Helper()

	data, err := scraperequestcontract.MarshalScrapeRequest(
		scraperequestcontract.ScrapeRequest{
			PageURL:  canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
			FetchURL: canonicalurltest.CanonicalURLOf(t, fetchURL),
		},
	)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return &pullintaketest.Message{Body: data}
}

func fetchOf(t *testing.T, text string) *fakeFetch {
	t.Helper()

	return &fakeFetch{outcome: pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			LandedURL:   canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
			ContentType: "text/html",
			Body: []byte(
				`<html lang="en"><head><title>` + pageTitle + `</title></head>` +
					`<body><p>` + text + `</p></body></html>`,
			),
		},
	}}
}

func run(
	t *testing.T,
	msg jetstream.Msg,
	fetcher scraperequestintake.PageFetcher,
	urls urlmeta.URLReceiver,
	postings rwiadmission.PostingReceiver,
) error {
	t.Helper()

	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("page formats: %v", err)
	}

	return scraperequestintake.NewScrapeRequestConsumer(scraperequestintake.Config{
		Source:                         pullintaketest.MessageSourceOf(msg),
		Fetcher:                        fetcher,
		FormatDerivations:              formatDerivations,
		URLs:                           urls,
		Postings:                       postings,
		ScrapeRequestIntakeConcurrency: 1,
	}).Run(context.Background())
}

func TestConsumerStoresTheIndexItDerivesFromAScrapeRequest(t *testing.T) {
	fetcher := fetchOf(t, "alpha beta")
	urls := &recordingURLs{}
	postings := &recordingPostings{}

	message := scrapeRequestMessage(t)
	if err := run(t, message, fetcher, urls, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(fetcher.urls) != 1 || fetcher.urls[0] != scrapeRequestURL {
		t.Errorf("fetched %v, want the scrape request url", fetcher.urls)
	}
	if len(urls.received) != 1 || urls.received[0].Address != scrapeRequestURL {
		t.Fatalf("stored metadata %+v, want one row for the scrape request", urls.received)
	}
	if urls.received[0].Title != pageTitle {
		t.Errorf("stored title = %q, want the extracted title", urls.received[0].Title)
	}
	assertWordsAdmitted(t, postings, "alpha", "beta")
}

func TestConsumerReadsAPageFromTheFetchURLAndStoresItUnderThePageURL(t *testing.T) {
	const replayURL = "http://archive.example/replay/https://example.com/"
	fetcher := fetchOf(t, "alpha beta")
	fetcher.outcome.Page.LandedURL = canonicalurltest.CanonicalURLOf(t, replayURL)
	urls := &recordingURLs{}
	postings := &recordingPostings{}

	message := archivedScrapeRequestMessage(t, replayURL)
	if err := run(t, message, fetcher, urls, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(fetcher.urls) != 1 || fetcher.urls[0] != replayURL {
		t.Errorf("fetched %v, want the fetch url", fetcher.urls)
	}
	if len(urls.received) != 1 || urls.received[0].Address != scrapeRequestURL {
		t.Fatalf("stored metadata %+v, want one row under the page url", urls.received)
	}
}

func TestConsumerAdmitsEveryPostingOfAPageInOneCall(t *testing.T) {
	postings := &recordingPostings{}

	if err := run(t, scrapeRequestMessage(t),
		fetchOf(t, "alpha beta gamma delta epsilon"),
		&recordingURLs{}, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	assertWordsAdmitted(t, postings, "alpha", "beta", "gamma", "delta", "epsilon")
}

func TestConsumerAcksAScrapeRequestHoldingNoExtractableDocument(t *testing.T) {
	postings := &recordingPostings{}
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			LandedURL:   canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
			ContentType: "application/pdf",
			Body:        []byte("%PDF-1.4"),
		},
	}}

	message := scrapeRequestMessage(t)
	if err := run(t, message,
		fetcher, &recordingURLs{}, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(postings.calls) != 0 {
		t.Errorf("stored %v, want nothing", postings.calls)
	}
}

func TestConsumerNaksWhenTheFetchBreaks(t *testing.T) {
	fetcher := &fakeFetch{err: errors.New("fetch broke down")}

	message := scrapeRequestMessage(t)
	if err := run(t, message,
		fetcher, &recordingURLs{}, &recordingPostings{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak", action)
	}
}

func TestConsumerNaksWhenTheFetchReportsFailure(t *testing.T) {
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}}

	message := scrapeRequestMessage(t)
	if err := run(t, message,
		fetcher, &recordingURLs{}, &recordingPostings{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak", action)
	}
}

func TestConsumerHoldsADeferredScrapeRequestBackForAsLongAsTheOriginAsks(t *testing.T) {
	message := scrapeRequestMessage(t)
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: 30 * time.Second,
	}}

	if err := run(t, message, fetcher, &recordingURLs{}, &recordingPostings{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak-with-delay", action)
	}
	if message.HeldBackFor() != 30*time.Second {
		t.Errorf("nak delay = %v, want the deferral the origin asked for", message.HeldBackFor())
	}
}

func TestConsumerAcksAScrapeRequestThatFetchesNoPage(t *testing.T) {
	postings := &recordingPostings{}
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchRejected}}

	message := scrapeRequestMessage(t)
	if err := run(t, message,
		fetcher, &recordingURLs{}, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(postings.calls) != 0 {
		t.Errorf("stored %v, want nothing", postings.calls)
	}
}

func TestConsumerNaksAndWithholdsPostingsWhenURLStorageIsBusy(t *testing.T) {
	postings := &recordingPostings{}

	message := scrapeRequestMessage(t)
	if err := run(t, message, fetchOf(t, "alpha"),
		&recordingURLs{receipt: urlmeta.Receipt{Busy: true}}, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak", action)
	}
	if len(postings.calls) != 0 {
		t.Errorf("stored %v, want no postings while url storage is busy", postings.calls)
	}
}

func TestConsumerNaksWhenPostingAdmissionRefuses(t *testing.T) {
	for name, postings := range map[string]*recordingPostings{
		"busy":    {receipt: rwiadmission.Receipt{Busy: true}},
		"failing": {err: errors.New("boom")},
	} {
		t.Run(name, func(t *testing.T) {
			message := scrapeRequestMessage(t)

			if err := run(t, message,
				fetchOf(t, "alpha"), &recordingURLs{}, postings); err != nil {
				t.Fatalf("run: %v", err)
			}

			if action := message.Settlement(t); action != pullintaketest.HeldBack {
				t.Errorf("action = %q, want nak", action)
			}
		})
	}
}

func TestConsumerHaltsOnAnUndecodableMessage(t *testing.T) {
	msg := &pullintaketest.Message{Body: []byte("not a scrape request")}

	err := run(t, msg, fetchOf(t, "alpha"), &recordingURLs{}, &recordingPostings{})

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("err = %v, want a poison message halt", err)
	}
	if settled := msg.Settlements(); len(settled) != 0 {
		t.Errorf("undecodable message settled %v, want it left pending", settled)
	}
}

func assertWordsAdmitted(t *testing.T, postings *recordingPostings, words ...string) {
	t.Helper()

	if len(postings.calls) != 1 {
		t.Fatalf("admitted over %d calls, want one call for the page", len(postings.calls))
	}
	admitted := map[yacymodel.Hash]bool{}
	for _, posting := range postings.calls[0] {
		admitted[posting.WordHash] = true
	}
	for _, word := range words {
		if !admitted[yacymodel.WordHash(word)] {
			t.Errorf("word %q should be admitted, got %v", word, postings.calls[0])
		}
	}
}
