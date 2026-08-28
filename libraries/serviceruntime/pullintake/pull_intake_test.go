package pullintake_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
)

func messages(count int) []jetstream.Msg {
	msgs := make([]jetstream.Msg, count)
	for i := range msgs {
		msgs[i] = &pullintaketest.Message{Body: []byte("payload")}
	}
	return msgs
}

func TestRunProcessesEveryMessageThenExitsOnClose(t *testing.T) {
	source := pullintaketest.MessageSourceOf(messages(3)...)
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
	source := pullintaketest.MessageSourceOf(messages(3)...)
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
	source := pullintaketest.MessageSourceHeldOnItsFirstMessage(release, messages(2)...)
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
	source := pullintaketest.MessageSourceThatCannotOpen(errors.New("no iterator"))
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
	source := pullintaketest.MessageSourceOf(msg)
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
	deliver(t, &pullintaketest.Message{Body: []byte("payload")},
		func(_ context.Context, message pullintake.PendingMessage) {
			body = message.Body()
		})

	if string(body) != "payload" {
		t.Errorf("body = %q, want the message payload", body)
	}
}

func TestPendingMessageIdentityNamesWhereTheMessageSits(t *testing.T) {
	var identity string
	deliver(t, &pullintaketest.Message{Stream: "YACY_SCRAPE_REQUEST", Sequence: 7},
		func(_ context.Context, message pullintake.PendingMessage) {
			identity = message.Identity()
		})

	if identity != pullintaketest.Subject+" YACY_SCRAPE_REQUEST/7" {
		t.Errorf("identity = %q, want the subject, stream and position", identity)
	}
}

func TestPendingMessageIdentityFallsBackToTheSubjectAlone(t *testing.T) {
	var identity string
	deliver(t, &pullintaketest.Message{MetadataError: errors.New("no metadata")},
		func(_ context.Context, message pullintake.PendingMessage) {
			identity = message.Identity()
		})

	if identity != pullintaketest.Subject {
		t.Errorf("identity = %q, want the subject alone", identity)
	}
}

func TestPendingMessageAcknowledgesTheMessage(t *testing.T) {
	msg := &pullintaketest.Message{}

	deliver(t, msg, func(ctx context.Context, message pullintake.PendingMessage) {
		message.Acknowledge(ctx)
	})

	if settled := msg.Settlements(); len(settled) != 1 ||
		settled[0] != pullintaketest.Acknowledged {
		t.Errorf("message settled %v, want one ack", settled)
	}
}

func TestPendingMessageHoldsAReturnedMessageBackBeforeItComesAgain(t *testing.T) {
	msg := &pullintaketest.Message{}

	deliver(t, msg, func(ctx context.Context, message pullintake.PendingMessage) {
		message.Return(ctx)
	})

	if settled := msg.Settlements(); len(settled) != 1 || settled[0] != pullintaketest.HeldBack {
		t.Errorf("message settled %v, want one delayed return", settled)
	}
	if msg.HeldBackFor() <= 0 {
		t.Errorf("delay = %v, want a pause before the message comes again", msg.HeldBackFor())
	}
}

func TestPendingMessageHoldsTheMessageBackForTheGivenDelay(t *testing.T) {
	msg := &pullintaketest.Message{}

	deliver(t, msg, func(ctx context.Context, message pullintake.PendingMessage) {
		message.ReturnAfter(ctx, 30*time.Second)
	})

	if msg.HeldBackFor() != 30*time.Second {
		t.Errorf("delay = %v, want the delay asked for", msg.HeldBackFor())
	}
}
