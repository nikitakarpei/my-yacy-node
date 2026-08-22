package scraperequestintake_test

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
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/scraperequestintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const (
	scrapeRequestURL = "https://example.com/"
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

func scrapedPage(t *testing.T, text string) *fakeScrape {
	t.Helper()
	return &fakeScrape{
		page: pagescrape.ScrapedPage{
			CanonicalURL: canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
			Title:        "Hi",
			Language:     "en",
			Content:      []byte(text),
		},
		scraped: true,
	}
}

func run(
	t *testing.T,
	msg jetstream.Msg,
	scraper scraperequestintake.PageScraper,
	urls urlmeta.URLReceiver,
	postings rwiadmission.PostingReceiver,
) error {
	t.Helper()

	return scraperequestintake.NewScrapeRequestConsumer(scraperequestintake.Config{
		Source:      fakeSource{iterator: &fakeIterator{messages: []jetstream.Msg{msg}}},
		Scraper:     scraper,
		URLs:        urls,
		Postings:    postings,
		Concurrency: 1,
	}).Run(context.Background())
}

func TestConsumerStoresTheIndexItDerivesFromAScrapeRequest(t *testing.T) {
	acked := make(chan string, 1)
	scraper := scrapedPage(t, "alpha beta")
	urls := &recordingURLs{}
	postings := &recordingPostings{}

	if err := run(t, scrapeRequestMessage(t, acked), scraper, urls, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(scraper.urls) != 1 || scraper.urls[0] != scrapeRequestURL {
		t.Errorf("scraped %v, want the scrape request url", scraper.urls)
	}
	if len(scraper.targetFormats) != 1 ||
		scraper.targetFormats[0] != documentextraction.FormatFullText {
		t.Errorf("scraped as %v, want %s", scraper.targetFormats, documentextraction.FormatFullText)
	}
	if len(urls.received) != 1 || urls.received[0].Address != scrapeRequestURL {
		t.Fatalf("stored metadata %+v, want one row for the scrape request", urls.received)
	}
	if urls.received[0].Title != "Hi" {
		t.Errorf("stored title = %q, want the scraped title", urls.received[0].Title)
	}
	if len(postings.calls) != 1 || len(postings.calls[0]) != 2 {
		t.Errorf("admitted %v, want one call carrying two postings", postings.calls)
	}
}

func TestConsumerAdmitsEveryPostingOfAPageInOneCall(t *testing.T) {
	acked := make(chan string, 1)
	postings := &recordingPostings{}

	if err := run(t, scrapeRequestMessage(t, acked),
		scrapedPage(t, "alpha beta gamma delta epsilon"),
		&recordingURLs{}, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	<-acked
	if len(postings.calls) != 1 {
		t.Fatalf("admitted over %d calls, want one call for the page", len(postings.calls))
	}
	if len(postings.calls[0]) != 5 {
		t.Fatalf("admitted %d postings, want one per distinct word", len(postings.calls[0]))
	}
}

func TestConsumerAcksAScrapeRequestThatDerivesNoIndex(t *testing.T) {
	acked := make(chan string, 1)
	postings := &recordingPostings{}

	if err := run(t, scrapeRequestMessage(t, acked),
		&fakeScrape{}, &recordingURLs{}, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(postings.calls) != 0 {
		t.Errorf("stored %v, want nothing", postings.calls)
	}
}

func TestConsumerNaksWhenTheScrapeFails(t *testing.T) {
	acked := make(chan string, 1)
	scraper := &fakeScrape{err: errors.New("fetch broke down")}

	if err := run(t, scrapeRequestMessage(t, acked),
		scraper, &recordingURLs{}, &recordingPostings{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak" {
		t.Errorf("action = %q, want nak", action)
	}
}

func TestConsumerNaksAndWithholdsPostingsWhenURLStorageIsBusy(t *testing.T) {
	acked := make(chan string, 1)
	postings := &recordingPostings{}

	if err := run(t, scrapeRequestMessage(t, acked), scrapedPage(t, "alpha"),
		&recordingURLs{receipt: urlmeta.Receipt{Busy: true}}, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak" {
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
			acked := make(chan string, 1)

			if err := run(t, scrapeRequestMessage(t, acked),
				scrapedPage(t, "alpha"), &recordingURLs{}, postings); err != nil {
				t.Fatalf("run: %v", err)
			}

			if action := <-acked; action != "nak" {
				t.Errorf("action = %q, want nak", action)
			}
		})
	}
}

func TestConsumerHaltsOnAnUndecodableMessage(t *testing.T) {
	acked := make(chan string, 1)
	msg := &fakeMsg{data: []byte("not a scrape request"), acked: acked}

	err := run(t, msg, scrapedPage(t, "alpha"), &recordingURLs{}, &recordingPostings{})

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("err = %v, want a poison message halt", err)
	}
	select {
	case action := <-acked:
		t.Errorf("message answered with %q, want it left pending", action)
	default:
	}
}
