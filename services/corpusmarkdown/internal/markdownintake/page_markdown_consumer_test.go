package markdownintake_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownintake"
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

type recordingCorpus struct {
	fail     bool
	mu       sync.Mutex
	markdown map[string][]byte
}

func (c *recordingCorpus) Put(_ context.Context, canonicalURL string, markdown []byte) error {
	if c.fail {
		return errors.New("store failed")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.markdown == nil {
		c.markdown = map[string][]byte{}
	}
	c.markdown[canonicalURL] = markdown
	return nil
}

type recordingProgress struct {
	received int
	stored   int
	failed   int
}

func (p *recordingProgress) PageReceived() { p.received++ }
func (p *recordingProgress) PageStored()   { p.stored++ }
func (p *recordingProgress) StoreFailed()  { p.failed++ }

func markdownMessage(
	t *testing.T,
	canonicalURL string,
	markdown []byte,
	acked chan string,
) *fakeMsg {
	t.Helper()
	data, err := yacycrawlcontract.MarshalPageMarkdownRepresentation(
		yacycrawlcontract.PageMarkdownRepresentation{
			PageReference: yacycrawlcontract.PageReference{CanonicalURL: canonicalURL},
			Markdown:      markdown,
		},
	)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return &fakeMsg{data: data, acked: acked}
}

func TestPageMarkdownConsumerStoresAndAcks(t *testing.T) {
	acked := make(chan string, 1)
	const canonicalURL = "https://example.com/"
	markdown := []byte("# Hi")
	source := fakeSource{iterator: &fakeIterator{
		messages: []jetstream.Msg{markdownMessage(t, canonicalURL, markdown, acked)},
	}}
	corpus := &recordingCorpus{}
	progress := &recordingProgress{}
	consumer := markdownintake.NewPageMarkdownConsumer(source, corpus, progress, 1)

	if err := consumer.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	got := corpus.markdown[canonicalURL]
	if string(got) != string(markdown) {
		t.Errorf("stored = %q, want %q", got, markdown)
	}
	if progress.received != 1 || progress.stored != 1 || progress.failed != 0 {
		t.Errorf("progress = %+v, want one received/stored", progress)
	}
}

func TestPageMarkdownConsumerNaksOnStoreFailure(t *testing.T) {
	acked := make(chan string, 1)
	source := fakeSource{iterator: &fakeIterator{
		messages: []jetstream.Msg{markdownMessage(t, "https://example.com/", []byte("x"), acked)},
	}}
	progress := &recordingProgress{}
	consumer := markdownintake.NewPageMarkdownConsumer(
		source,
		&recordingCorpus{fail: true},
		progress,
		1,
	)

	if err := consumer.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if action := <-acked; action != "nak" {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.received != 1 || progress.stored != 0 || progress.failed != 1 {
		t.Errorf("progress = %+v, want one received/failed", progress)
	}
}

func TestPageMarkdownConsumerHaltsOnDecodeFailure(t *testing.T) {
	acked := make(chan string, 1)
	source := fakeSource{iterator: &fakeIterator{
		messages: []jetstream.Msg{&fakeMsg{data: []byte("not json"), acked: acked}},
	}}
	progress := &recordingProgress{}
	consumer := markdownintake.NewPageMarkdownConsumer(source, &recordingCorpus{}, progress, 1)

	err := consumer.Run(context.Background())
	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("run error = %v, want poison halt", err)
	}
	select {
	case action := <-acked:
		t.Fatalf("undecodable message was %q, want left pending", action)
	default:
	}
	if progress.received != 1 || progress.stored != 0 || progress.failed != 0 {
		t.Errorf("progress = %+v, want one received and no store outcome", progress)
	}
}
