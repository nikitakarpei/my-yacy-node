package consumerbacklog_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/admissions/consumerbacklog"
)

const (
	streamName   = "PAGES"
	subject      = "pages.offered"
	reader       = "reader"
	backlogLimit = 2
	namespace    = "service"
	heldBack     = namespace + "_intake_held_back"
)

type clock struct {
	mu       sync.Mutex
	now      time.Time
	sleeping chan struct{}
	wake     chan struct{}
}

func clockThatNeverSleeps() *clock {
	return &clock{now: time.Unix(0, 0)}
}

func clockFiredByTheTest() *clock {
	return &clock{now: time.Unix(0, 0), sleeping: make(chan struct{}), wake: make(chan struct{})}
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Sleep(ctx context.Context, d time.Duration) error {
	if c.wake == nil {
		return errors.New("the intake was held back")
	}
	c.sleeping <- struct{}{}
	select {
	case <-c.wake:
		c.mu.Lock()
		c.now = c.now.Add(d)
		c.mu.Unlock()
		return nil
	case <-ctx.Done():
		return fmt.Errorf("sleep: %w", ctx.Err())
	}
}

type pages struct {
	js     jetstream.JetStream
	stream jetstream.Stream
}

func pagesStream(t *testing.T) pages {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	stream, err := js.CreateStream(t.Context(), jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
	})
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}
	return pages{js: js, stream: stream}
}

func (p pages) offer(t *testing.T, amount int) {
	t.Helper()
	for range amount {
		if _, err := p.js.Publish(t.Context(), subject, []byte("page")); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}
}

func (p pages) readerOf(t *testing.T, durable string) jetstream.Consumer {
	t.Helper()
	consumer, err := p.stream.CreateConsumer(t.Context(), jetstream.ConsumerConfig{
		Durable:   durable,
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create consumer: %v", err)
	}
	return consumer
}

func takeAll(t *testing.T, consumer jetstream.Consumer, amount int) {
	t.Helper()
	batch, err := consumer.Fetch(amount, jetstream.FetchMaxWait(5*time.Second))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	for msg := range batch.Messages() {
		if err := msg.DoubleAck(t.Context()); err != nil {
			t.Fatalf("ack: %v", err)
		}
	}
}

func takeWithoutAcknowledging(t *testing.T, consumer jetstream.Consumer, amount int) {
	t.Helper()
	batch, err := consumer.Fetch(amount, jetstream.FetchMaxWait(5*time.Second))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	for range batch.Messages() {
	}
}

func heldBackBy(t *testing.T, registry *prometheusclient.Registry, consumer string) float64 {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, family := range families {
		if family.GetName() != heldBack {
			continue
		}
		for _, metric := range family.GetMetric() {
			if metric.GetLabel()[0].GetValue() == consumer {
				return metric.GetGauge().GetValue()
			}
		}
	}
	return 0
}

func admissionOf(
	stream consumerbacklog.Stream,
	c *clock,
) (*consumerbacklog.Admission, *prometheusclient.Registry) {
	registry := prometheusclient.NewRegistry()
	return consumerbacklog.New(stream, backlogLimit, c, registry, namespace), registry
}

func TestAdmitsWhileEveryBacklogIsWithinTheLimit(t *testing.T) {
	pages := pagesStream(t)
	pages.readerOf(t, reader)
	pages.offer(t, backlogLimit)
	admission, _ := admissionOf(pages.stream, clockThatNeverSleeps())

	if err := admission.AwaitAdmission(t.Context()); err != nil {
		t.Fatalf("await admission: %v", err)
	}
}

func TestHoldsBackUntilTheConsumerCatchesUp(t *testing.T) {
	pages := pagesStream(t)
	consumer := pages.readerOf(t, reader)
	pages.offer(t, backlogLimit+1)
	c := clockFiredByTheTest()
	admission, registry := admissionOf(pages.stream, c)
	admitted := make(chan error, 1)

	go func() { admitted <- admission.AwaitAdmission(t.Context()) }()
	<-c.sleeping
	if got := heldBackBy(t, registry, reader); got != 1 {
		t.Errorf("%s{consumer=%q} = %v while held back, want 1", heldBack, reader, got)
	}
	takeAll(t, consumer, backlogLimit+1)
	c.wake <- struct{}{}

	if err := <-admitted; err != nil {
		t.Fatalf("await admission: %v", err)
	}
	if got := heldBackBy(t, registry, reader); got != 0 {
		t.Errorf("%s{consumer=%q} = %v after the catch-up, want 0", heldBack, reader, got)
	}
}

func TestHoldsBackWhileTakenMessagesAreNotFinished(t *testing.T) {
	pages := pagesStream(t)
	consumer := pages.readerOf(t, reader)
	pages.offer(t, backlogLimit+1)
	takeWithoutAcknowledging(t, consumer, backlogLimit+1)
	admission, _ := admissionOf(pages.stream, clockThatNeverSleeps())

	if err := admission.AwaitAdmission(t.Context()); err == nil {
		t.Fatal("await admission admitted while the reader had not finished its messages")
	}
}

func TestIgnoresConsumersWithoutADurableName(t *testing.T) {
	pages := pagesStream(t)
	pages.readerOf(t, "")
	pages.offer(t, backlogLimit+1)
	admission, _ := admissionOf(pages.stream, clockThatNeverSleeps())

	if err := admission.AwaitAdmission(t.Context()); err != nil {
		t.Fatalf("await admission: %v", err)
	}
}

func TestStopsAwaitingWhenTheContextEnds(t *testing.T) {
	pages := pagesStream(t)
	pages.readerOf(t, reader)
	pages.offer(t, backlogLimit+1)
	c := clockFiredByTheTest()
	admission, _ := admissionOf(pages.stream, c)
	ctx, cancel := context.WithCancel(t.Context())
	admitted := make(chan error, 1)

	go func() { admitted <- admission.AwaitAdmission(ctx) }()
	<-c.sleeping
	cancel()

	if err := <-admitted; !errors.Is(err, context.Canceled) {
		t.Fatalf("await admission = %v, want the cancellation", err)
	}
}

type unreadableStream struct{}

func (unreadableStream) ListConsumers(context.Context) jetstream.ConsumerInfoLister {
	return unreadableConsumers{}
}

type unreadableConsumers struct{}

func (unreadableConsumers) Info() <-chan *jetstream.ConsumerInfo {
	infos := make(chan *jetstream.ConsumerInfo)
	close(infos)
	return infos
}

func (unreadableConsumers) Err() error {
	return errors.New("no responders")
}

func TestAdmitsWhenTheBacklogsCannotBeRead(t *testing.T) {
	admission, _ := admissionOf(unreadableStream{}, clockThatNeverSleeps())

	if err := admission.AwaitAdmission(t.Context()); err != nil {
		t.Fatalf("await admission: %v", err)
	}
}
