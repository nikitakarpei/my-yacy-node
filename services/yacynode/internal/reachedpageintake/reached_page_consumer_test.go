package reachedpageintake_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/reachedpageintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const (
	reachedPageURL  = "https://example.com/"
	postingBatchCap = 2
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
	batches [][]yacymodel.RWIPosting
}

func (r *recordingPostings) Receive(
	_ context.Context,
	postings []yacymodel.RWIPosting,
) (rwiadmission.Receipt, error) {
	r.batches = append(r.batches, postings)

	return r.receipt, r.err
}

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

func scrapedPage(text string) *fakeScrape {
	return &fakeScrape{
		page: pagescrape.ScrapedPage{
			CanonicalURL: reachedPageURL,
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
	scraper reachedpageintake.PageScraper,
	urls urlmeta.URLReceiver,
	postings rwiadmission.PostingReceiver,
) error {
	t.Helper()

	return reachedpageintake.NewReachedPageConsumer(reachedpageintake.Config{
		Source:          fakeSource{iterator: &fakeIterator{messages: []jetstream.Msg{msg}}},
		Scraper:         scraper,
		URLs:            urls,
		Postings:        postings,
		PostingBatchCap: postingBatchCap,
		Concurrency:     1,
	}).Run(context.Background())
}

func TestConsumerStoresTheIndexItDerivesFromAReachedPage(t *testing.T) {
	acked := make(chan string, 1)
	scraper := scrapedPage("alpha beta")
	urls := &recordingURLs{}
	postings := &recordingPostings{}

	if err := run(t, reachedPageMessage(t, acked), scraper, urls, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(scraper.urls) != 1 || scraper.urls[0] != reachedPageURL {
		t.Errorf("scraped %v, want the reached page url", scraper.urls)
	}
	if len(urls.received) != 1 || urls.received[0].Address != reachedPageURL {
		t.Fatalf("stored metadata %+v, want one row for the reached page", urls.received)
	}
	if urls.received[0].Title != "Hi" {
		t.Errorf("stored title = %q, want the scraped title", urls.received[0].Title)
	}
	if len(postings.batches) != 1 || len(postings.batches[0]) != 2 {
		t.Errorf("stored batches %v, want one batch of two postings", postings.batches)
	}
}

func TestConsumerStoresPostingsInBatchesTheAdmissionAccepts(t *testing.T) {
	acked := make(chan string, 1)
	postings := &recordingPostings{}

	if err := run(t, reachedPageMessage(t, acked),
		scrapedPage("alpha beta gamma delta epsilon"),
		&recordingURLs{}, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	<-acked
	if len(postings.batches) != 3 {
		t.Fatalf(
			"stored %d batches, want five postings split by a cap of two",
			len(postings.batches),
		)
	}
	for _, batch := range postings.batches {
		if len(batch) > postingBatchCap {
			t.Errorf("batch of %d postings exceeds the cap of %d", len(batch), postingBatchCap)
		}
	}
}

func TestConsumerAcksAReachedPageThatDerivesNoIndex(t *testing.T) {
	acked := make(chan string, 1)
	postings := &recordingPostings{}

	if err := run(t, reachedPageMessage(t, acked),
		&fakeScrape{}, &recordingURLs{}, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(postings.batches) != 0 {
		t.Errorf("stored %v, want nothing", postings.batches)
	}
}

func TestConsumerAcksAReachedPageWhoseURLNoIndexCanBeBuiltFrom(t *testing.T) {
	acked := make(chan string, 1)
	scraper := scrapedPage("alpha")
	scraper.page.CanonicalURL = "://nonsense"
	postings := &recordingPostings{}

	if err := run(
		t,
		reachedPageMessage(t, acked),
		scraper,
		&recordingURLs{},
		postings,
	); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(postings.batches) != 0 {
		t.Errorf("stored %v, want nothing", postings.batches)
	}
}

func TestConsumerNaksWhenTheScrapeFails(t *testing.T) {
	acked := make(chan string, 1)
	scraper := &fakeScrape{err: errors.New("fetch broke down")}

	if err := run(t, reachedPageMessage(t, acked),
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

	if err := run(t, reachedPageMessage(t, acked), scrapedPage("alpha"),
		&recordingURLs{receipt: urlmeta.Receipt{Busy: true}}, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak" {
		t.Errorf("action = %q, want nak", action)
	}
	if len(postings.batches) != 0 {
		t.Errorf("stored %v, want no postings while url storage is busy", postings.batches)
	}
}

func TestConsumerNaksWhenPostingAdmissionRefuses(t *testing.T) {
	for name, postings := range map[string]*recordingPostings{
		"busy":      {receipt: rwiadmission.Receipt{Busy: true}},
		"too large": {receipt: rwiadmission.Receipt{Busy: true, TooLarge: true}},
		"failing":   {err: errors.New("boom")},
	} {
		t.Run(name, func(t *testing.T) {
			acked := make(chan string, 1)

			if err := run(t, reachedPageMessage(t, acked),
				scrapedPage("alpha"), &recordingURLs{}, postings); err != nil {
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
	msg := &fakeMsg{data: []byte("not a reached page"), acked: acked}

	err := run(t, msg, scrapedPage("alpha"), &recordingURLs{}, &recordingPostings{})

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("err = %v, want a poison message halt", err)
	}
	select {
	case action := <-acked:
		t.Errorf("message answered with %q, want it left pending", action)
	default:
	}
}
