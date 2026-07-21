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

type fakeMsg struct{ data []byte }

func (m *fakeMsg) Subject() string                  { return "subj" }
func (m *fakeMsg) Reply() string                    { return "" }
func (m *fakeMsg) Data() []byte                     { return m.data }
func (m *fakeMsg) Headers() nats.Header             { return nil }
func (m *fakeMsg) Ack() error                       { return nil }
func (m *fakeMsg) DoubleAck(context.Context) error  { return nil }
func (m *fakeMsg) Nak() error                       { return nil }
func (m *fakeMsg) NakWithDelay(time.Duration) error { return nil }
func (m *fakeMsg) InProgress() error                { return nil }
func (m *fakeMsg) Term() error                      { return nil }
func (m *fakeMsg) TermWithReason(string) error      { return nil }

func (m *fakeMsg) Metadata() (*jetstream.MsgMetadata, error) {
	return &jetstream.MsgMetadata{}, nil
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
		func(context.Context, jetstream.Msg) error {
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
		func(context.Context, jetstream.Msg) error {
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
		done <- pullintake.Run(ctx, source, 1, func(context.Context, jetstream.Msg) error {
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
		func(context.Context, jetstream.Msg) error {
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected error when source cannot open")
	}
}
