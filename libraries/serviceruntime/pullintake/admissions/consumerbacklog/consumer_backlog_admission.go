// Package consumerbacklog admits intake while every durable consumer of one stream
// has at most a set number of messages left to finish, read or not, so a service
// takes in work only as fast as the slowest reader of what it publishes.
package consumerbacklog

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

const (
	backlogCheckInterval = 2 * time.Second

	labelConsumer = "consumer"

	msgIntakeHeldBack   = "intake held back by a consumer backlog"
	msgIntakeResumed    = "intake resumed"
	msgBacklogsUnread   = "consumer backlogs not read, intake admitted"
	noConsumerHoldsBack = ""
)

type Stream interface {
	ListConsumers(ctx context.Context) jetstream.ConsumerInfoLister
}

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type Admission struct {
	stream           Stream
	backlogLimit     uint64
	clock            Clock
	holdingConsumers *prometheusclient.GaugeVec
	checkedAt        time.Time
	holdingConsumer  string
}

func New(
	stream Stream,
	backlogLimit uint64,
	clock Clock,
	registry prometheusclient.Registerer,
	serviceMetricNamespace string,
) *Admission {
	admission := &Admission{
		stream:       stream,
		backlogLimit: backlogLimit,
		clock:        clock,
		holdingConsumers: prometheusclient.NewGaugeVec(
			prometheusclient.GaugeOpts{
				Name: serviceMetricNamespace + "_intake_held_back",
				Help: "1 while the backlog of this consumer holds the intake back.",
			},
			[]string{labelConsumer},
		),
	}
	registry.MustRegister(admission.holdingConsumers)
	return admission
}

func (a *Admission) AwaitAdmission(ctx context.Context) error {
	for a.heldBack(ctx) {
		if err := a.clock.Sleep(ctx, backlogCheckInterval); err != nil {
			return fmt.Errorf("await admission: %w", err)
		}
	}
	return nil
}

func (a *Admission) heldBack(ctx context.Context) bool {
	now := a.clock.Now()
	if now.Sub(a.checkedAt) >= backlogCheckInterval {
		a.checkedAt = now
		largest, err := a.largestBacklog(ctx)
		a.settle(ctx, largest, err)
	}
	return a.holdingConsumer != noConsumerHoldsBack
}

type consumerBacklog struct {
	consumer string
	amount   uint64
}

func (a *Admission) largestBacklog(ctx context.Context) (consumerBacklog, error) {
	consumers := a.stream.ListConsumers(ctx)
	var largest consumerBacklog
	for info := range consumers.Info() {
		amount := info.NumPending + uint64(info.NumAckPending) //nolint:gosec // never negative
		if info.Config.Durable != "" && amount > largest.amount {
			largest = consumerBacklog{consumer: info.Name, amount: amount}
		}
	}
	if err := consumers.Err(); err != nil {
		return largest, fmt.Errorf("list consumers: %w", err)
	}
	return largest, nil
}

func (a *Admission) settle(ctx context.Context, largest consumerBacklog, err error) {
	switch {
	case err != nil:
		slog.WarnContext(ctx, msgBacklogsUnread, slog.Any("error", err))
		a.release(ctx)
	case largest.amount <= a.backlogLimit:
		a.release(ctx)
	case largest.consumer != a.holdingConsumer:
		a.holdBackBy(ctx, largest)
	}
}

func (a *Admission) holdBackBy(ctx context.Context, largest consumerBacklog) {
	slog.WarnContext(ctx, msgIntakeHeldBack,
		slog.String(labelConsumer, largest.consumer),
		slog.Uint64("backlog", largest.amount),
		slog.Uint64("limit", a.backlogLimit),
	)
	if a.holdingConsumer != noConsumerHoldsBack {
		a.holdingConsumers.WithLabelValues(a.holdingConsumer).Set(0)
	}
	a.holdingConsumers.WithLabelValues(largest.consumer).Set(1)
	a.holdingConsumer = largest.consumer
}

func (a *Admission) release(ctx context.Context) {
	if a.holdingConsumer == noConsumerHoldsBack {
		return
	}
	slog.DebugContext(ctx, msgIntakeResumed, slog.String(labelConsumer, a.holdingConsumer))
	a.holdingConsumers.WithLabelValues(a.holdingConsumer).Set(0)
	a.holdingConsumer = noConsumerHoldsBack
}
