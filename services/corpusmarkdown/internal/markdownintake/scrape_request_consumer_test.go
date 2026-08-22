package markdownintake_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownintake"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
)

type fakeMsg struct {
	data  []byte
	acked chan string
}

func (m *fakeMsg) Subject() string                  { return "subj" }
func (m *fakeMsg) Reply() string                    { return "" }
func (m *fakeMsg) Data() []byte                     { return m.data }
func (m *fakeMsg) Headers() nats.Header             { return nil }
func (m *fakeMsg) Ack() error                       { m.acked <- "ack"; return nil }
func (m *fakeMsg) DoubleAck(context.Context) error  { m.acked <- "ack"; return nil }
func (m *fakeMsg) Nak() error                       { m.acked <- "nak"; return nil }
func (m *fakeMsg) NakWithDelay(time.Duration) error { m.acked <- "nak"; return nil }
func (m *fakeMsg) InProgress() error                { return nil }
func (m *fakeMsg) Term() error                      { m.acked <- "term"; return nil }
func (m *fakeMsg) TermWithReason(string) error      { m.acked <- "term"; return nil }

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

type fakeScrape struct {
	page          pagescrape.ScrapedPage
	scraped       bool
	err           error
	mu            sync.Mutex
	urls          []string
	targetFormats []documentextraction.Format
}

func (s *fakeScrape) Scrape(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
	targetFormat documentextraction.Format,
) (pagescrape.ScrapedPage, bool, error) {
	s.mu.Lock()
	s.urls = append(s.urls, pageURL.String())
	s.targetFormats = append(s.targetFormats, targetFormat)
	s.mu.Unlock()
	return s.page, s.scraped, s.err
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

func (p *recordingProgress) PageReceived() { p.received++ }
func (p *recordingProgress) PageStored()   { p.stored++ }
func (p *recordingProgress) ScrapeFailed() { p.scraped++ }
func (p *recordingProgress) StoreFailed()  { p.failed++ }

const scrapeRequestURL = "https://example.com/"

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

func run(
	t *testing.T,
	source fakeSource,
	scraper markdownintake.PageScraper,
	corpus markdownintake.PageMarkdownCorpus,
	progress markdownintake.StoreProgress,
) error {
	t.Helper()
	return markdownintake.NewScrapeRequestConsumer(source, scraper, corpus, progress, 1).
		Run(context.Background())
}

func TestConsumerStoresTheMarkdownItScrapesFromAScrapeRequest(t *testing.T) {
	acked := make(chan string, 1)
	scraper := &fakeScrape{
		page: pagescrape.ScrapedPage{
			CanonicalURL: canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
			Content:      []byte("# Hi"),
		},
		scraped: true,
	}
	corpus := &recordingCorpus{}
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		scraper, corpus, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(scraper.urls) != 1 || scraper.urls[0] != scrapeRequestURL {
		t.Errorf("scraped %v, want the scrape request url", scraper.urls)
	}
	if len(scraper.targetFormats) != 1 ||
		scraper.targetFormats[0] != documentextraction.FormatMarkdown {
		t.Errorf("scraped as %v, want %s", scraper.targetFormats, documentextraction.FormatMarkdown)
	}
	if got := corpus.markdown[scrapeRequestURL]; string(got) != "# Hi" {
		t.Errorf("stored = %q, want # Hi", got)
	}
	if progress.received != 1 || progress.stored != 1 {
		t.Errorf("progress = %+v, want one received and one stored", progress)
	}
}

func TestConsumerAcksAScrapeRequestThatDerivesNoMarkdown(t *testing.T) {
	acked := make(chan string, 1)
	corpus := &recordingCorpus{}
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		&fakeScrape{}, corpus, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(corpus.markdown) != 0 || progress.stored != 0 {
		t.Errorf("stored %v, want nothing", corpus.markdown)
	}
}

func TestConsumerNaksWhenTheScrapeFails(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		&fakeScrape{err: errors.New("fetch broke")}, &recordingCorpus{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak" {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.scraped != 1 || progress.stored != 0 {
		t.Errorf("progress = %+v, want one scrape failure", progress)
	}
}

func TestConsumerNaksWhenTheStoreFails(t *testing.T) {
	acked := make(chan string, 1)
	scraper := &fakeScrape{
		page: pagescrape.ScrapedPage{
			CanonicalURL: canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
			Content:      []byte("x"),
		},
		scraped: true,
	}
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		scraper, &recordingCorpus{fail: true}, progress); err != nil {
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
		&fakeScrape{}, &recordingCorpus{}, progress)

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
