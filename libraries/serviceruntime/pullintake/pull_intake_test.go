package pullintake_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
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

func TestRunFailsWhenTheSourceFails(t *testing.T) {
	want := errors.New("consumer deleted")
	source := pullintaketest.MessageSourceThatFails(want)
	err := pullintake.Run(
		context.Background(),
		source,
		1,
		func(context.Context, pullintake.PendingMessage) error {
			return nil
		},
	)
	if !errors.Is(err, want) {
		t.Fatalf("run error = %v, want %v", err, want)
	}
}

func TestRunTakesAMessageOnlyWhenAWorkerIsFree(t *testing.T) {
	source := pullintaketest.MessageSourceOf(messages(3)...)
	working := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- pullintake.Run(context.Background(), source, 1,
			func(context.Context, pullintake.PendingMessage) error {
				working <- struct{}{}
				<-release
				return nil
			})
	}()
	<-working
	if taken := source.AmountOfTakenMessages(); taken != 1 {
		t.Errorf("took %d messages while the only worker was busy, want 1", taken)
	}
	close(release)
	<-working
	<-working
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}
}

func consumerOfPublished(t *testing.T, amount int) jetstream.Consumer {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	stream, err := js.CreateStream(t.Context(), jetstream.StreamConfig{
		Name:     "WORK",
		Subjects: []string{"work"},
	})
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}
	for range amount {
		if _, err := js.Publish(t.Context(), "work", []byte("payload")); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}
	consumer, err := stream.CreateConsumer(t.Context(), jetstream.ConsumerConfig{
		Durable:   "worker",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create consumer: %v", err)
	}
	return consumer
}

func TestRunTakesEveryBrokerMessageThenWaitsUntilTheContextEnds(t *testing.T) {
	consumer := consumerOfPublished(t, 2)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	processed := make(chan struct{}, 2)
	done := make(chan error, 1)
	go func() {
		done <- pullintake.Run(ctx, consumer, 1,
			func(ctx context.Context, message pullintake.PendingMessage) error {
				message.Acknowledge(ctx)
				processed <- struct{}{}
				return nil
			})
	}()
	<-processed
	<-processed
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}
}

type contextPastItsDeadline struct {
	context.Context
}

func (contextPastItsDeadline) Deadline() (time.Time, bool) {
	return time.Unix(0, 0), true
}

func TestRunTakesBrokerMessagesUntilAContextPastItsDeadlineEnds(t *testing.T) {
	consumer := consumerOfPublished(t, 2)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	processed := make(chan struct{}, 2)
	done := make(chan error, 1)
	go func() {
		done <- pullintake.Run(contextPastItsDeadline{ctx}, consumer, 1,
			func(ctx context.Context, message pullintake.PendingMessage) error {
				message.Acknowledge(ctx)
				processed <- struct{}{}
				return nil
			})
	}()
	for range 2 {
		select {
		case <-processed:
		case err := <-done:
			t.Fatalf("run ended before it took every message: %v", err)
		}
	}
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}
}

type admissionOf struct {
	admitted int
}

func (a *admissionOf) AwaitAdmission(ctx context.Context) error {
	if a.admitted == 0 {
		<-ctx.Done()
		return fmt.Errorf("await admission: %w", ctx.Err())
	}
	a.admitted--
	return nil
}

func TestRunAdmittedByTakesOnlyAdmittedMessages(t *testing.T) {
	source := pullintaketest.MessageSourceOf(messages(3)...)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	processed := make(chan struct{}, 3)
	done := make(chan error, 1)
	go func() {
		done <- pullintake.RunAdmittedBy(ctx, source, 1, &admissionOf{admitted: 2},
			func(context.Context, pullintake.PendingMessage) error {
				processed <- struct{}{}
				return nil
			})
	}()
	<-processed
	<-processed
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}
	if extra := len(processed); extra != 0 {
		t.Errorf("processed %d messages past the admission", extra)
	}
	if taken := source.AmountOfTakenMessages(); taken != 2 {
		t.Errorf("took %d messages, want only the 2 admitted", taken)
	}
}

func TestRunAdmittedByHaltsOnFatalErrorWhileHeldBack(t *testing.T) {
	source := pullintaketest.MessageSourceOf(messages(2)...)
	want := errors.New("poison")
	err := pullintake.RunAdmittedBy(context.Background(), source, 1, &admissionOf{admitted: 1},
		func(context.Context, pullintake.PendingMessage) error {
			return want
		})
	if !errors.Is(err, want) {
		t.Fatalf("run error = %v, want %v", err, want)
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

func TestPendingMessageArrivalIsWhenTheMessageEnteredItsStream(t *testing.T) {
	entered := time.Date(2026, time.October, 11, 9, 0, 0, 0, time.UTC)
	var arrival time.Time
	deliver(t, &pullintaketest.Message{Arrival: entered},
		func(_ context.Context, message pullintake.PendingMessage) {
			arrival = message.Arrival()
		})

	if !arrival.Equal(entered) {
		t.Errorf("arrival = %v, want %v", arrival, entered)
	}
}

func TestPendingMessageArrivalIsUnknownWithoutMetadata(t *testing.T) {
	var arrival time.Time
	deliver(t, &pullintaketest.Message{MetadataError: errors.New("no metadata")},
		func(_ context.Context, message pullintake.PendingMessage) {
			arrival = message.Arrival()
		})

	if !arrival.IsZero() {
		t.Errorf("arrival = %v, want none", arrival)
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
