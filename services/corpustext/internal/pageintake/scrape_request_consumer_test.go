package pageintake_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageintake"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
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

type recordingIndex struct {
	fail      bool
	mu        sync.Mutex
	documents []searchdocument.Document
}

func (i *recordingIndex) Index(_ context.Context, document searchdocument.Document) error {
	if i.fail {
		return errors.New("index failed")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.documents = append(i.documents, document)
	return nil
}

type recordingProgress struct {
	received       int
	indexed        int
	scrapeFailures int
	indexFailures  int
	observed       int
}

func (p *recordingProgress) ScrapeRequestReceived()      { p.received++ }
func (p *recordingProgress) PageIndexed()                { p.indexed++ }
func (p *recordingProgress) ScrapeFailed()               { p.scrapeFailures++ }
func (p *recordingProgress) IndexFailed()                { p.indexFailures++ }
func (p *recordingProgress) IndexObserved(time.Duration) { p.observed++ }

const scrapeRequestURL = "https://example.com/"

const article = `<!DOCTYPE html><html lang="en"><head><title>Sample Article</title></head>` +
	`<body><article><h1>Sample Article</h1><p>` + longText + `</p><p>` + longText +
	`</p></article></body></html>`

const longText = "The quick brown fox jumps over the lazy dog while the industrious " +
	"beaver builds a sturdy dam across the wide and winding river near the old mill town."

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

func fetchOf(page pagefetch.FetchedPage) *fakeFetch {
	return &fakeFetch{
		outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchSucceeded, Page: page},
	}
}

func fetchedPage(t *testing.T, contentType string, body string) pagefetch.FetchedPage {
	t.Helper()
	return pagefetch.FetchedPage{
		LandedURL:   canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
		ContentType: contentType,
		Body:        []byte(body),
	}
}

func run(
	t *testing.T,
	source pullintaketest.MessageSource,
	fetcher pageintake.PageFetcher,
	searchIndex pageintake.SearchIndex,
	progress pageintake.IndexProgress,
) error {
	t.Helper()
	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("page formats: %v", err)
	}
	return pageintake.NewScrapeRequestConsumer(pageintake.Config{
		Source:                         source,
		Fetcher:                        fetcher,
		FormatDerivations:              formatDerivations,
		SearchIndex:                    searchIndex,
		Progress:                       progress,
		ScrapeRequestIntakeConcurrency: 1,
	}).Run(context.Background())
}

func TestConsumerIndexesTheTextItDerivesFromAScrapeRequest(t *testing.T) {
	message := scrapeRequestMessage(t)
	fetcher := fetchOf(fetchedPage(t, "text/html", article))
	searchIndex := &recordingIndex{}
	progress := &recordingProgress{}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		fetcher, searchIndex, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(fetcher.urls) != 1 || fetcher.urls[0] != scrapeRequestURL {
		t.Errorf("fetched %v, want the scrape request url", fetcher.urls)
	}
	if len(searchIndex.documents) != 1 {
		t.Fatalf("indexed %v, want one document", searchIndex.documents)
	}
	indexed := searchIndex.documents[0]
	if indexed.URL != scrapeRequestURL || indexed.Title != "Sample Article" ||
		indexed.Language != "en" || !strings.Contains(indexed.Content, "quick brown fox") {
		t.Errorf("indexed = %+v, want the article as readable text", indexed)
	}
	if progress.received != 1 || progress.indexed != 1 || progress.observed != 1 {
		t.Errorf("progress = %+v, want one received/indexed/observed", progress)
	}
}

func TestConsumerReadsAPageFromTheFetchURLAndIndexesItUnderThePageURL(t *testing.T) {
	const replayURL = "http://archive.example/replay/https://example.com/"
	message := archivedScrapeRequestMessage(t, replayURL)
	page := fetchedPage(t, "text/html", article)
	page.LandedURL = canonicalurltest.CanonicalURLOf(t, replayURL)
	fetcher := fetchOf(page)
	searchIndex := &recordingIndex{}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		fetcher, searchIndex, &recordingProgress{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(fetcher.urls) != 1 || fetcher.urls[0] != replayURL {
		t.Errorf("fetched %v, want the fetch url", fetcher.urls)
	}
	if len(searchIndex.documents) != 1 || searchIndex.documents[0].URL != scrapeRequestURL {
		t.Fatalf("indexed %v, want one document under the page url", searchIndex.documents)
	}
}

func TestConsumerAcksAScrapeRequestHoldingNoExtractableDocument(t *testing.T) {
	message := scrapeRequestMessage(t)
	searchIndex := &recordingIndex{}
	progress := &recordingProgress{}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		fetchOf(fetchedPage(t, "application/pdf", "%PDF-1.4")),
		searchIndex, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(searchIndex.documents) != 0 || progress.indexed != 0 {
		t.Errorf("indexed %v, want nothing", searchIndex.documents)
	}
}

func TestConsumerNaksWhenTheFetchBreaks(t *testing.T) {
	message := scrapeRequestMessage(t)
	progress := &recordingProgress{}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		&fakeFetch{err: errors.New("fetch broke")}, &recordingIndex{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.scrapeFailures != 1 || progress.observed != 0 {
		t.Errorf("progress = %+v, want one scrape failure and no index attempt", progress)
	}
}

func TestConsumerNaksWhenTheFetchReportsFailure(t *testing.T) {
	message := scrapeRequestMessage(t)
	progress := &recordingProgress{}
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		fetcher, &recordingIndex{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.scrapeFailures != 1 {
		t.Errorf("progress = %+v, want one scrape failure", progress)
	}
}

func TestConsumerHoldsADeferredScrapeRequestBackForAsLongAsTheOriginAsks(t *testing.T) {
	message := scrapeRequestMessage(t)
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: 30 * time.Second,
	}}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		fetcher, &recordingIndex{}, &recordingProgress{}); err != nil {
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
	message := scrapeRequestMessage(t)
	progress := &recordingProgress{}
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchRejected}}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		fetcher, &recordingIndex{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if progress.indexed != 0 || progress.scrapeFailures != 0 {
		t.Errorf("progress = %+v, want nothing indexed and no failure", progress)
	}
}

func TestConsumerNaksWhenTheIndexFails(t *testing.T) {
	message := scrapeRequestMessage(t)
	progress := &recordingProgress{}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		fetchOf(fetchedPage(t, "text/html", article)),
		&recordingIndex{fail: true}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.indexFailures != 1 || progress.indexed != 0 || progress.observed != 1 {
		t.Errorf("progress = %+v, want one index failure", progress)
	}
}

func TestConsumerHaltsOnAnUndecodableMessage(t *testing.T) {
	message := &pullintaketest.Message{Body: []byte("not json")}
	progress := &recordingProgress{}

	err := run(t, pullintaketest.MessageSourceOf(message),
		&fakeFetch{}, &recordingIndex{}, progress)

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("run error = %v, want poison halt", err)
	}
	if settled := message.Settlements(); len(settled) != 0 {
		t.Fatalf("undecodable message settled %v, want left pending", settled)
	}
	if progress.received != 1 {
		t.Errorf("progress = %+v, want one received", progress)
	}
}
