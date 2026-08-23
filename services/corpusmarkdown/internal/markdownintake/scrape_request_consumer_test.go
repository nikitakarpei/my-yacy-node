package markdownintake_test

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
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownintake"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
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

type recordingCorpus struct {
	fail     bool
	mu       sync.Mutex
	markdown map[string][]byte
}

func (c *recordingCorpus) Put(
	_ context.Context,
	scrapeRequestURL canonicalurl.CanonicalURL,
	markdown []byte,
) error {
	if c.fail {
		return errors.New("store failed")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.markdown == nil {
		c.markdown = map[string][]byte{}
	}
	c.markdown[scrapeRequestURL.String()] = markdown
	return nil
}

type recordingProgress struct {
	received int
	stored   int
	scraped  int
	failed   int
}

func (p *recordingProgress) ScrapeRequestReceived() { p.received++ }
func (p *recordingProgress) PageStored()            { p.stored++ }
func (p *recordingProgress) ScrapeFailed()          { p.scraped++ }
func (p *recordingProgress) StoreFailed()           { p.failed++ }

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
			CanonicalURL: canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
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
		FinalURL:    canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
		ContentType: contentType,
		Body:        []byte(body),
	}
}

func run(
	t *testing.T,
	source fakeSource,
	fetcher markdownintake.PageFetcher,
	corpus markdownintake.PageMarkdownCorpus,
	progress markdownintake.StoreProgress,
) error {
	t.Helper()
	derivableFormats, err := pageformats.New()
	if err != nil {
		t.Fatalf("page formats: %v", err)
	}
	return markdownintake.NewScrapeRequestConsumer(markdownintake.Config{
		Source:      source,
		Fetcher:     fetcher,
		Formats:     derivableFormats,
		Corpus:      corpus,
		Progress:    progress,
		Concurrency: 1,
	}).Run(context.Background())
}

func TestConsumerStoresTheMarkdownItDerivesFromAScrapeRequest(t *testing.T) {
	acked := make(chan string, 1)
	fetcher := fetchOf(fetchedPage(t, "text/html", article))
	corpus := &recordingCorpus{}
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetcher, corpus, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(fetcher.urls) != 1 || fetcher.urls[0] != scrapeRequestURL {
		t.Errorf("fetched %v, want the scrape request url", fetcher.urls)
	}
	if got := string(corpus.markdown[scrapeRequestURL]); !strings.Contains(got, "quick brown fox") {
		t.Errorf("stored = %q, want the article as markdown", got)
	}
	if progress.received != 1 || progress.stored != 1 {
		t.Errorf("progress = %+v, want one received and one stored", progress)
	}
}

func TestConsumerAcksAScrapeRequestHoldingNoExtractableDocument(t *testing.T) {
	acked := make(chan string, 1)
	corpus := &recordingCorpus{}
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetchOf(fetchedPage(t, "application/pdf", "%PDF-1.4")), corpus, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(corpus.markdown) != 0 || progress.stored != 0 {
		t.Errorf("stored %v, want nothing", corpus.markdown)
	}
}

func TestConsumerNaksWhenTheFetchBreaks(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		&fakeFetch{err: errors.New("fetch broke")}, &recordingCorpus{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak" {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.scraped != 1 || progress.stored != 0 {
		t.Errorf("progress = %+v, want one scrape failure", progress)
	}
}

func TestConsumerNaksWhenTheFetchReportsFailure(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetcher, &recordingCorpus{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak" {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.scraped != 1 {
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
		fetcher, &recordingCorpus{}, &recordingProgress{}); err != nil {
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
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchNotAPage}}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetcher, &recordingCorpus{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if progress.stored != 0 || progress.scraped != 0 {
		t.Errorf("progress = %+v, want nothing stored and no failure", progress)
	}
}

func TestConsumerNaksWhenTheStoreFails(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetchOf(fetchedPage(t, "text/html", article)),
		&recordingCorpus{fail: true}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak" {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.failed != 1 || progress.stored != 0 {
		t.Errorf("progress = %+v, want one store failure", progress)
	}
}

func TestConsumerHaltsOnAnUndecodableMessage(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	err := run(t, sourceOf(&fakeMsg{data: []byte("not json"), acked: acked}),
		&fakeFetch{}, &recordingCorpus{}, progress)

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
