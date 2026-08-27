package pageintake_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageintake"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
)

type fakeMsg struct {
	data     []byte
	acked    chan string
	nakDelay time.Duration
}

func (m *fakeMsg) Subject() string                 { return "subj" }
func (m *fakeMsg) Reply() string                   { return "" }
func (m *fakeMsg) Data() []byte                    { return m.data }
func (m *fakeMsg) Headers() nats.Header            { return nil }
func (m *fakeMsg) Ack() error                      { m.acked <- "ack"; return nil }
func (m *fakeMsg) DoubleAck(context.Context) error { m.acked <- "ack"; return nil }
func (m *fakeMsg) Nak() error                      { m.acked <- "nak"; return nil }
func (m *fakeMsg) InProgress() error               { return nil }
func (m *fakeMsg) Term() error                     { m.acked <- "term"; return nil }
func (m *fakeMsg) TermWithReason(string) error     { m.acked <- "term"; return nil }

func (m *fakeMsg) NakWithDelay(delay time.Duration) error {
	m.nakDelay = delay
	m.acked <- "nak-with-delay"
	return nil
}

func (m *fakeMsg) Metadata() (*jetstream.MsgMetadata, error) {
	return &jetstream.MsgMetadata{}, nil
}

type fakeIterator struct {
	messages []jetstream.Msg
	mu       sync.Mutex
}

func (it *fakeIterator) Next(...jetstream.NextOpt) (jetstream.Msg, error) {
	it.mu.Lock()
	defer it.mu.Unlock()
	if len(it.messages) == 0 {
		return nil, jetstream.ErrMsgIteratorClosed
	}
	msg := it.messages[0]
	it.messages = it.messages[1:]
	return msg, nil
}

func (it *fakeIterator) Stop()  {}
func (it *fakeIterator) Drain() {}

type fakeSource struct {
	iterator *fakeIterator
}

func (s fakeSource) Messages(...jetstream.PullMessagesOpt) (jetstream.MessagesContext, error) {
	return s.iterator, nil
}

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

func scrapeRequestMessage(t *testing.T, acked chan string) *fakeMsg {
	t.Helper()
	data, err := scraperequestcontract.MarshalScrapeRequest(
		scraperequestcontract.ScrapeRequest{
			PageURL: canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
		},
	)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return &fakeMsg{data: data, acked: acked}
}

func archivedScrapeRequestMessage(t *testing.T, acked chan string, fetchURL string) *fakeMsg {
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
	return &fakeMsg{data: data, acked: acked}
}

func sourceOf(messages ...jetstream.Msg) fakeSource {
	return fakeSource{iterator: &fakeIterator{messages: messages}}
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
	source fakeSource,
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
	acked := make(chan string, 1)
	fetcher := fetchOf(fetchedPage(t, "text/html", article))
	searchIndex := &recordingIndex{}
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetcher, searchIndex, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
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
	acked := make(chan string, 1)
	page := fetchedPage(t, "text/html", article)
	page.LandedURL = canonicalurltest.CanonicalURLOf(t, replayURL)
	fetcher := fetchOf(page)
	searchIndex := &recordingIndex{}

	if err := run(t, sourceOf(archivedScrapeRequestMessage(t, acked, replayURL)),
		fetcher, searchIndex, &recordingProgress{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
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
	acked := make(chan string, 1)
	searchIndex := &recordingIndex{}
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetchOf(fetchedPage(t, "application/pdf", "%PDF-1.4")),
		searchIndex, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(searchIndex.documents) != 0 || progress.indexed != 0 {
		t.Errorf("indexed %v, want nothing", searchIndex.documents)
	}
}

func TestConsumerNaksWhenTheFetchBreaks(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		&fakeFetch{err: errors.New("fetch broke")}, &recordingIndex{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak-with-delay" {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.scrapeFailures != 1 || progress.observed != 0 {
		t.Errorf("progress = %+v, want one scrape failure and no index attempt", progress)
	}
}

func TestConsumerNaksWhenTheFetchReportsFailure(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetcher, &recordingIndex{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak-with-delay" {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.scrapeFailures != 1 {
		t.Errorf("progress = %+v, want one scrape failure", progress)
	}
}

func TestConsumerHoldsADeferredScrapeRequestBackForAsLongAsTheOriginAsks(t *testing.T) {
	acked := make(chan string, 1)
	message := scrapeRequestMessage(t, acked)
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: 30 * time.Second,
	}}

	if err := run(t, sourceOf(message),
		fetcher, &recordingIndex{}, &recordingProgress{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak-with-delay" {
		t.Errorf("action = %q, want nak-with-delay", action)
	}
	if message.nakDelay != 30*time.Second {
		t.Errorf("nak delay = %v, want the deferral the origin asked for", message.nakDelay)
	}
}

func TestConsumerAcksAScrapeRequestThatFetchesNoPage(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchRejected}}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetcher, &recordingIndex{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if progress.indexed != 0 || progress.scrapeFailures != 0 {
		t.Errorf("progress = %+v, want nothing indexed and no failure", progress)
	}
}

func TestConsumerNaksWhenTheIndexFails(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetchOf(fetchedPage(t, "text/html", article)),
		&recordingIndex{fail: true}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak-with-delay" {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.indexFailures != 1 || progress.indexed != 0 || progress.observed != 1 {
		t.Errorf("progress = %+v, want one index failure", progress)
	}
}

func TestConsumerHaltsOnAnUndecodableMessage(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	err := run(t, sourceOf(&fakeMsg{data: []byte("not json"), acked: acked}),
		&fakeFetch{}, &recordingIndex{}, progress)

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("run error = %v, want poison halt", err)
	}
	select {
	case action := <-acked:
		t.Fatalf("undecodable message was %q, want left pending", action)
	default:
	}
	if progress.received != 1 {
		t.Errorf("progress = %+v, want one received", progress)
	}
}
