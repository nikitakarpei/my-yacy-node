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
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
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

type recordingScrapeProgress struct {
	attemptOutcomes     []string
	urlMetadataAdmitted int
	postingsAdmitted    int
	postingsNotAdmitted []int
	admissionFailures   []error
}

func (r *recordingScrapeProgress) ScrapeRequestInvalid(context.Context) {
	r.attemptOutcomes = append(r.attemptOutcomes, "invalid_message")
}

func (r *recordingScrapeProgress) OriginFetchFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	error,
) {
	r.attemptOutcomes = append(r.attemptOutcomes, "fetch_failed")
}

func (r *recordingScrapeProgress) OriginFetchDeferred(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	time.Duration,
) {
	r.attemptOutcomes = append(r.attemptOutcomes, "fetch_deferred")
}

func (r *recordingScrapeProgress) NothingToScrape(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.attemptOutcomes = append(r.attemptOutcomes, "nothing_to_scrape")
}

func (r *recordingScrapeProgress) DocumentExtractionFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	error,
) {
	r.attemptOutcomes = append(r.attemptOutcomes, "document_extraction_failed")
}

func (r *recordingScrapeProgress) NoIndexDerived(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.attemptOutcomes = append(r.attemptOutcomes, "no_index_derived")
}

func (r *recordingScrapeProgress) URLMetadataAdmitted(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.urlMetadataAdmitted++
}

func (r *recordingScrapeProgress) URLMetadataAdmissionBusy(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.attemptOutcomes = append(r.attemptOutcomes, "url_metadata_admission_busy")
}

func (r *recordingScrapeProgress) URLMetadataAdmissionFailed(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	cause error,
) {
	r.attemptOutcomes = append(r.attemptOutcomes, "url_metadata_admission_failed")
	r.admissionFailures = append(r.admissionFailures, cause)
}

func (r *recordingScrapeProgress) PostingsAdmitted(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	postings int,
) {
	r.postingsAdmitted += postings
}

func (r *recordingScrapeProgress) PostingsAdmissionBusy(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	postings int,
) {
	r.attemptOutcomes = append(r.attemptOutcomes, "postings_admission_busy")
	r.postingsNotAdmitted = append(r.postingsNotAdmitted, postings)
}

func (r *recordingScrapeProgress) PostingsAdmissionFailed(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	postings int,
	cause error,
) {
	r.attemptOutcomes = append(r.attemptOutcomes, "postings_admission_failed")
	r.postingsNotAdmitted = append(r.postingsNotAdmitted, postings)
	r.admissionFailures = append(r.admissionFailures, cause)
}

func (r *recordingScrapeProgress) ScrapeRequestCompleted(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.attemptOutcomes = append(r.attemptOutcomes, "completed")
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

	data, err := pagescrapecontract.MarshalScrapeRequest(
		pagescrapecontract.ScrapeRequest{
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

	data, err := pagescrapecontract.MarshalScrapeRequest(
		pagescrapecontract.ScrapeRequest{
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
	return runWithScrapeProgress(
		t,
		msg,
		scrapeRequestConsumerCollaborators{
			pageFetcher:     fetcher,
			urlReceiver:     urls,
			postingReceiver: postings,
			scrapeProgress:  scraperequestintake.ScrapeProgressObservers{},
		},
	)
}

type scrapeRequestConsumerCollaborators struct {
	pageFetcher     scraperequestintake.PageFetcher
	urlReceiver     urlmeta.URLReceiver
	postingReceiver rwiadmission.PostingReceiver
	scrapeProgress  scraperequestintake.ScrapeProgress
}

func runWithScrapeProgress(
	t *testing.T,
	msg jetstream.Msg,
	collaborators scrapeRequestConsumerCollaborators,
) error {
	t.Helper()

	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("page formats: %v", err)
	}

	return scraperequestintake.NewScrapeRequestConsumer(
		scraperequestintake.ScrapeRequestConsumerConfig{
			ScrapeRequestSource:            pullintaketest.MessageSourceOf(msg),
			PageFetcher:                    collaborators.pageFetcher,
			FormatDerivations:              formatDerivations,
			URLReceiver:                    collaborators.urlReceiver,
			PostingReceiver:                collaborators.postingReceiver,
			ScrapeProgress:                 collaborators.scrapeProgress,
			ScrapeRequestIntakeConcurrency: 1,
		}).Run(context.Background())
}

func TestConsumerReportsTheCompletedAttemptAndEveryAdmission(t *testing.T) {
	progress := &recordingScrapeProgress{}
	postings := &recordingPostings{}

	if err := runWithScrapeProgress(
		t,
		scrapeRequestMessage(t),
		scrapeRequestConsumerCollaborators{
			pageFetcher:     fetchOf(t, "alpha beta"),
			urlReceiver:     &recordingURLs{},
			postingReceiver: postings,
			scrapeProgress:  progress,
		},
	); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := progress.attemptOutcomes; len(got) != 1 || got[0] != "completed" {
		t.Errorf("attempt outcomes = %v, want completed", got)
	}
	if progress.urlMetadataAdmitted != 1 {
		t.Errorf("url metadata admitted = %d, want 1", progress.urlMetadataAdmitted)
	}
	if progress.postingsAdmitted != len(postings.calls[0]) {
		t.Errorf("postings admitted = %d, want %d",
			progress.postingsAdmitted, len(postings.calls[0]))
	}
}

type admissionAttemptExpectation struct {
	urls             *recordingURLs
	postings         *recordingPostings
	wantOutcome      string
	wantURLMetadata  int
	wantPostingCalls int
	wantFailureCause bool
}

func TestConsumerReportsWhichAdmissionReturnedTheScrapeRequest(t *testing.T) {
	cause := errors.New("admission failed")
	for name, expectation := range map[string]admissionAttemptExpectation{
		"url busy": {
			urls:        &recordingURLs{receipt: urlmeta.Receipt{Busy: true}},
			postings:    &recordingPostings{},
			wantOutcome: "url_metadata_admission_busy",
		},
		"url failed": {
			urls:             &recordingURLs{err: cause},
			postings:         &recordingPostings{},
			wantOutcome:      "url_metadata_admission_failed",
			wantFailureCause: true,
		},
		"url rejected": {
			urls: &recordingURLs{receipt: urlmeta.Receipt{
				ErrorURL: []yacymodel.URLHash{{}},
			}},
			postings:         &recordingPostings{},
			wantOutcome:      "url_metadata_admission_failed",
			wantFailureCause: true,
		},
		"posting busy": {
			urls:             &recordingURLs{},
			postings:         &recordingPostings{receipt: rwiadmission.Receipt{Busy: true}},
			wantOutcome:      "postings_admission_busy",
			wantURLMetadata:  1,
			wantPostingCalls: 1,
		},
		"posting failed": {
			urls:             &recordingURLs{},
			postings:         &recordingPostings{err: cause},
			wantOutcome:      "postings_admission_failed",
			wantURLMetadata:  1,
			wantPostingCalls: 1,
			wantFailureCause: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			progress := &recordingScrapeProgress{}

			if err := runWithScrapeProgress(
				t,
				scrapeRequestMessage(t),
				scrapeRequestConsumerCollaborators{
					pageFetcher:     fetchOf(t, "alpha"),
					urlReceiver:     expectation.urls,
					postingReceiver: expectation.postings,
					scrapeProgress:  progress,
				},
			); err != nil {
				t.Fatalf("run: %v", err)
			}

			assertAdmissionAttempt(t, progress, expectation)
		})
	}
}

func assertAdmissionAttempt(
	t *testing.T,
	progress *recordingScrapeProgress,
	expectation admissionAttemptExpectation,
) {
	t.Helper()

	if got := progress.attemptOutcomes; len(got) != 1 || got[0] != expectation.wantOutcome {
		t.Errorf("attempt outcomes = %v, want %s", got, expectation.wantOutcome)
	}
	if progress.urlMetadataAdmitted != expectation.wantURLMetadata {
		t.Errorf("url metadata admitted = %d, want %d",
			progress.urlMetadataAdmitted, expectation.wantURLMetadata)
	}
	if progress.postingsAdmitted != 0 {
		t.Errorf("postings admitted = %d, want 0", progress.postingsAdmitted)
	}
	if len(expectation.postings.calls) != expectation.wantPostingCalls {
		t.Errorf("posting admission calls = %d, want %d",
			len(expectation.postings.calls), expectation.wantPostingCalls)
	}
	if expectation.wantPostingCalls == 0 && len(progress.postingsNotAdmitted) != 0 {
		t.Errorf("postings not admitted = %v, want none", progress.postingsNotAdmitted)
	}
	if expectation.wantPostingCalls == 1 &&
		(len(progress.postingsNotAdmitted) != 1 ||
			progress.postingsNotAdmitted[0] != len(expectation.postings.calls[0])) {
		t.Errorf("postings not admitted = %v, want %d",
			progress.postingsNotAdmitted, len(expectation.postings.calls[0]))
	}
	if got := len(progress.admissionFailures) != 0; got != expectation.wantFailureCause {
		t.Errorf("failure cause reported = %t, want %t", got, expectation.wantFailureCause)
	}
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

func TestConsumerNaksWhenPostingsAdmissionRefuses(t *testing.T) {
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
