package pageintake_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageintake"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
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
	page    pagescrape.ScrapedPage
	scraped bool
	err     error
	mu      sync.Mutex
	urls    []string
}

func (s *fakeScrape) Scrape(
	_ context.Context,
	pageURL string,
) (pagescrape.ScrapedPage, bool, error) {
	s.mu.Lock()
	s.urls = append(s.urls, pageURL)
	s.mu.Unlock()
	return s.page, s.scraped, s.err
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

func (p *recordingProgress) PageReceived()               { p.received++ }
func (p *recordingProgress) PageIndexed()                { p.indexed++ }
func (p *recordingProgress) ScrapeFailed()               { p.scrapeFailures++ }
func (p *recordingProgress) IndexFailed()                { p.indexFailures++ }
func (p *recordingProgress) IndexObserved(time.Duration) { p.observed++ }

const reachedPageURL = "https://example.com/"

func reachedPageMessage(t *testing.T, acked chan string) *fakeMsg {
	t.Helper()
	data, err := yacycrawlcontract.MarshalReachedPage(
		yacycrawlcontract.ReachedPage{CanonicalURL: reachedPageURL},
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
	scraper pageintake.PageScraper,
	searchIndex pageintake.SearchIndex,
	progress pageintake.IndexProgress,
) error {
	t.Helper()
	return pageintake.NewReachedPageConsumer(source, scraper, searchIndex, progress, 1).
		Run(context.Background())
}

func scrapedPage() *fakeScrape {
	return &fakeScrape{
		page: pagescrape.ScrapedPage{
			CanonicalURL: reachedPageURL,
			Title:        "Hi",
			Language:     "en",
			Content:      []byte("words here"),
		},
		scraped: true,
	}
}

func TestConsumerIndexesTheTextItScrapesFromAReachedPage(t *testing.T) {
	acked := make(chan string, 1)
	scraper := scrapedPage()
	searchIndex := &recordingIndex{}
	progress := &recordingProgress{}

	if err := run(t, sourceOf(reachedPageMessage(t, acked)),
		scraper, searchIndex, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(scraper.urls) != 1 || scraper.urls[0] != reachedPageURL {
		t.Errorf("scraped %v, want the reached page url", scraper.urls)
	}
	if len(searchIndex.documents) != 1 {
		t.Fatalf("indexed %v, want one document", searchIndex.documents)
	}
	indexed := searchIndex.documents[0]
	if indexed.URL != reachedPageURL || indexed.Title != "Hi" ||
		indexed.Content != "words here" || indexed.Language != "en" {
		t.Errorf("indexed = %+v, want the scraped page", indexed)
	}
	if progress.received != 1 || progress.indexed != 1 || progress.observed != 1 {
		t.Errorf("progress = %+v, want one received/indexed/observed", progress)
	}
}

func TestConsumerAcksAReachedPageThatDerivesNoText(t *testing.T) {
	acked := make(chan string, 1)
	searchIndex := &recordingIndex{}
	progress := &recordingProgress{}

	if err := run(t, sourceOf(reachedPageMessage(t, acked)),
		&fakeScrape{}, searchIndex, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(searchIndex.documents) != 0 || progress.indexed != 0 {
		t.Errorf("indexed %v, want nothing", searchIndex.documents)
	}
}

func TestConsumerNaksWhenTheScrapeFails(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	if err := run(t, sourceOf(reachedPageMessage(t, acked)),
		&fakeScrape{err: errors.New("fetch broke")}, &recordingIndex{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak" {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.scrapeFailures != 1 || progress.observed != 0 {
		t.Errorf("progress = %+v, want one scrape failure and no index attempt", progress)
	}
}

func TestConsumerNaksWhenTheIndexFails(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	if err := run(t, sourceOf(reachedPageMessage(t, acked)),
		scrapedPage(), &recordingIndex{fail: true}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak" {
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
		&fakeScrape{}, &recordingIndex{}, progress)

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
