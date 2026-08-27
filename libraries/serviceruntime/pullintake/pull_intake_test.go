package pullintake_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
)

type fakeMsg struct {
	data       []byte
	metadata   *jetstream.MsgMetadata
	metaErr    error
	settlement chan string
	nakDelay   time.Duration
}

func (m *fakeMsg) Subject() string                 { return "subj" }
func (m *fakeMsg) Reply() string                   { return "" }
func (m *fakeMsg) Data() []byte                    { return m.data }
func (m *fakeMsg) Headers() nats.Header            { return nil }
func (m *fakeMsg) Ack() error                      { m.settle("ack"); return nil }
func (m *fakeMsg) DoubleAck(context.Context) error { return nil }
func (m *fakeMsg) Nak() error                      { m.settle("nak"); return nil }

func (m *fakeMsg) InProgress() error           { return nil }
func (m *fakeMsg) Term() error                 { return nil }
func (m *fakeMsg) TermWithReason(string) error { return nil }

func (m *fakeMsg) NakWithDelay(delay time.Duration) error {
	m.nakDelay = delay
	m.settle("nak-with-delay")
	return nil
}

func (m *fakeMsg) Metadata() (*jetstream.MsgMetadata, error) {
	if m.metaErr != nil {
		return nil, m.metaErr
	}
	if m.metadata != nil {
		return m.metadata, nil
	}
	return &jetstream.MsgMetadata{}, nil
}

func (m *fakeMsg) settle(action string) {
	if m.settlement != nil {
		m.settlement <- action
	}
}

type fakeIterator struct {
	mu       sync.Mutex
	messages []jetstream.Msg
	blockOn  int
	release  chan struct{}
}

func (it *fakeIterator) Next(...jetstream.NextOpt) (jetstream.Msg, error) {
	it.mu.Lock()
	if len(it.messages) == 0 {
		it.mu.Unlock()
		return nil, jetstream.ErrMsgIteratorClosed
	}
	msg := it.messages[0]
	it.messages = it.messages[1:]
	release := it.release
	block := it.blockOn > 0
	it.blockOn--
	it.mu.Unlock()
	if block && release != nil {
		<-release
	}
	return msg, nil
}

func (it *fakeIterator) Stop()  {}
func (it *fakeIterator) Drain() {}

type fakeSource struct {
	iterator *fakeIterator
	openErr  error
}

func (s fakeSource) Messages(...jetstream.PullMessagesOpt) (jetstream.MessagesContext, error) {
	if s.openErr != nil {
		return nil, s.openErr
	}
	return s.iterator, nil
}

func messages(count int) []jetstream.Msg {
	msgs := make([]jetstream.Msg, count)
	for i := range msgs {
		msgs[i] = &fakeMsg{data: []byte("payload")}
	}
	return msgs
}

func TestRunProcessesEveryMessageThenExitsOnClose(t *testing.T) {
	source := fakeSource{iterator: &fakeIterator{messages: messages(3)}}
	var mu sync.Mutex
	seen := 0
	err := pullintake.Run(
		context.Background(),
		source,
		2,
		func(context.Context, pullintake.PendingMessage) error {
			mu.Lock()
			seen++
			mu.Unlock()
			return nil
		},
	)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if seen != 3 {
		t.Errorf("processed %d messages, want 3", seen)
	}
}

func TestRunHaltsOnFirstFatalError(t *testing.T) {
	source := fakeSource{iterator: &fakeIterator{messages: messages(3)}}
	want := errors.New("poison")
	err := pullintake.Run(
		context.Background(),
		source,
		1,
		func(context.Context, pullintake.PendingMessage) error {
			return want
		},
	)
	if !errors.Is(err, want) {
		t.Fatalf("run error = %v, want %v", err, want)
	}
}

func TestRunReturnsNilOnContextCancel(t *testing.T) {
	release := make(chan struct{})
	source := fakeSource{iterator: &fakeIterator{
		messages: messages(2),
		blockOn:  1,
		release:  release,
	}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- pullintake.Run(ctx, source, 1, func(context.Context, pullintake.PendingMessage) error {
			return nil
		})
	}()
	cancel()
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after cancel")
	}
}

func TestRunFailsWhenSourceCannotOpen(t *testing.T) {
	source := fakeSource{openErr: errors.New("no iterator")}
	err := pullintake.Run(
		context.Background(),
		source,
		1,
		func(context.Context, pullintake.PendingMessage) error {
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected error when source cannot open")
	}
}

func deliver(
	t *testing.T,
	msg jetstream.Msg,
	act func(ctx context.Context, message pullintake.PendingMessage),
) {
	t.Helper()
	source := fakeSource{iterator: &fakeIterator{messages: []jetstream.Msg{msg}}}
	err := pullintake.Run(
		context.Background(),
		source,
		1,
		func(ctx context.Context, message pullintake.PendingMessage) error {
			act(ctx, message)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestPendingMessageCarriesTheMessageBody(t *testing.T) {
	var body []byte
	deliver(t, &fakeMsg{data: []byte("payload")},
		func(_ context.Context, message pullintake.PendingMessage) {
			body = message.Body()
		})

	if string(body) != "payload" {
		t.Errorf("body = %q, want the message payload", body)
	}
}

func TestPendingMessageIdentityNamesWhereTheMessageSits(t *testing.T) {
	var identity string
	deliver(t, &fakeMsg{metadata: &jetstream.MsgMetadata{
		Stream:   "YACY_SCRAPE_REQUEST",
		Sequence: jetstream.SequencePair{Stream: 7},
	}}, func(_ context.Context, message pullintake.PendingMessage) {
		identity = message.Identity()
	})

	if identity != "subj YACY_SCRAPE_REQUEST/7" {
		t.Errorf("identity = %q, want the subject, stream and position", identity)
	}
}

func TestPendingMessageCountsItsDeliveries(t *testing.T) {
	var deliveries uint64
	deliver(t, &fakeMsg{metadata: &jetstream.MsgMetadata{NumDelivered: 3}},
		func(_ context.Context, message pullintake.PendingMessage) {
			deliveries = message.Deliveries()
		})

	if deliveries != 3 {
		t.Errorf("deliveries = %d, want the count the broker keeps", deliveries)
	}
}

func TestPendingMessageWithoutMetadataCountsOneDelivery(t *testing.T) {
	var deliveries uint64
	deliver(t, &fakeMsg{metaErr: errors.New("no metadata")},
		func(_ context.Context, message pullintake.PendingMessage) {
			deliveries = message.Deliveries()
		})

	if deliveries != 1 {
		t.Errorf("deliveries = %d, want one", deliveries)
	}
}

func TestPendingMessageIdentityFallsBackToTheSubjectAlone(t *testing.T) {
	var identity string
	deliver(t, &fakeMsg{metaErr: errors.New("no metadata")},
		func(_ context.Context, message pullintake.PendingMessage) {
			identity = message.Identity()
		})

	if identity != "subj" {
		t.Errorf("identity = %q, want the subject alone", identity)
	}
}

func TestPendingMessageAcknowledgesTheMessage(t *testing.T) {
	msg := &fakeMsg{settlement: make(chan string, 1)}

	deliver(t, msg, func(ctx context.Context, message pullintake.PendingMessage) {
		message.Acknowledge(ctx)
	})

	if action := <-msg.settlement; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
}

func TestPendingMessageReturnsTheMessageForRedelivery(t *testing.T) {
	msg := &fakeMsg{settlement: make(chan string, 1)}

	deliver(t, msg, func(ctx context.Context, message pullintake.PendingMessage) {
		message.Return(ctx)
	})

	if action := <-msg.settlement; action != "nak" {
		t.Errorf("action = %q, want nak", action)
	}
}

func TestPendingMessageHoldsTheMessageBackForTheGivenDelay(t *testing.T) {
	msg := &fakeMsg{settlement: make(chan string, 1)}

	deliver(t, msg, func(ctx context.Context, message pullintake.PendingMessage) {
		message.ReturnAfter(ctx, 30*time.Second)
	})

	if action := <-msg.settlement; action != "nak-with-delay" {
		t.Errorf("action = %q, want nak-with-delay", action)
	}
	if msg.nakDelay != 30*time.Second {
		t.Errorf("delay = %v, want the delay asked for", msg.nakDelay)
	}
}
