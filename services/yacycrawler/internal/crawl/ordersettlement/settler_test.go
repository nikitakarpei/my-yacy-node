package ordersettlement_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/ordersettlement"
)

type fakeTraversal struct {
	err   error
	calls int
	block <-chan struct{}
}

func (f *fakeTraversal) Traverse(ctx context.Context, _ yacycrawlcontract.CrawlOrder) error {
	f.calls++
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return fmt.Errorf("fake traversal: %w", ctx.Err())
		}
	}
	return f.err
}

type recordingObserver struct {
	received    int
	completed   int
	redelivered int
}

func (o *recordingObserver) OrderReceived()    { o.received++ }
func (o *recordingObserver) OrderCompleted()   { o.completed++ }
func (o *recordingObserver) OrderRedelivered() { o.redelivered++ }

type manualClock struct{}

func (manualClock) Now() time.Time { return time.Time{} }

func (manualClock) Sleep(ctx context.Context, _ time.Duration) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("manual clock: %w", err)
	}
	return nil
}

type fakeDelivery struct {
	acknowledged   int
	returned       int
	acknowledgeErr error
	extend         func(context.Context) error
}

func (fakeDelivery) Order() yacycrawlcontract.CrawlOrder {
	return yacycrawlcontract.CrawlOrder{OrderID: "o1"}
}

func (d *fakeDelivery) Acknowledge(context.Context) error {
	d.acknowledged++
	return d.acknowledgeErr
}

func (d *fakeDelivery) Return(context.Context) error {
	d.returned++
	return nil
}

func (d *fakeDelivery) ExtendOwnership(ctx context.Context) error {
	if d.extend == nil {
		return nil
	}
	return d.extend(ctx)
}

func deliver(
	t *testing.T,
	settler *ordersettlement.Settler,
	delivery ordersettlement.OrderDelivery,
) {
	t.Helper()
	deliveries := make(chan ordersettlement.OrderDelivery, 1)
	deliveries <- delivery
	close(deliveries)
	if err := settler.Settle(context.Background(), deliveries); err != nil {
		t.Fatalf("settle: %v", err)
	}
}

func TestSettleAcknowledgesAndCompletesOnSuccess(t *testing.T) {
	observer := &recordingObserver{}
	delivery := &fakeDelivery{}

	deliver(t, ordersettlement.New(&fakeTraversal{}, observer, manualClock{}, 0), delivery)
	if delivery.acknowledged != 1 || observer.completed != 1 || observer.received != 1 {
		t.Fatalf("expected acknowledge+complete: %+v observer=%+v", delivery, observer)
	}
}

func TestSettleRedeliversOnTraversalError(t *testing.T) {
	observer := &recordingObserver{}
	delivery := &fakeDelivery{}

	deliver(
		t,
		ordersettlement.New(&fakeTraversal{err: errors.New("boom")}, observer, manualClock{}, 0),
		delivery,
	)
	if delivery.returned != 1 || delivery.acknowledged != 0 ||
		observer.redelivered != 1 || observer.completed != 0 {
		t.Fatalf("expected redelivery: %+v observer=%+v", delivery, observer)
	}
}

func TestSettleToleratesAcknowledgeError(t *testing.T) {
	observer := &recordingObserver{}
	delivery := &fakeDelivery{acknowledgeErr: errors.New("acknowledge failed")}

	deliver(t, ordersettlement.New(&fakeTraversal{}, observer, manualClock{}, 0), delivery)
	if observer.completed != 0 {
		t.Fatalf("acknowledge failure should not count as completed: %+v", observer)
	}
}

func TestSettleStopsWhenDeliveriesClose(t *testing.T) {
	deliveries := make(chan ordersettlement.OrderDelivery)
	close(deliveries)

	if err := ordersettlement.New(
		&fakeTraversal{}, &recordingObserver{}, manualClock{}, 0,
	).Settle(context.Background(), deliveries); err != nil {
		t.Fatalf("settle: %v", err)
	}
}

func TestSettleStopsWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := ordersettlement.New(
		&fakeTraversal{},
		&recordingObserver{},
		manualClock{},
		0,
	).Settle(ctx, make(chan ordersettlement.OrderDelivery)); err == nil {
		t.Fatal("cancelled context should end the run with an error")
	}
}

func TestSettleRenewsOwnershipWhileTraversing(t *testing.T) {
	gate := make(chan struct{})
	traversal := &fakeTraversal{block: gate}

	var openOnce sync.Once
	var renewed atomic.Int64
	extend := func(context.Context) error {
		renewed.Add(1)
		openOnce.Do(func() { close(gate) })
		return nil
	}

	deliver(
		t,
		ordersettlement.New(traversal, &recordingObserver{}, &tickingClock{}, time.Millisecond),
		&fakeDelivery{extend: extend},
	)

	if renewed.Load() == 0 {
		t.Fatal("expected ownership heartbeat to extend at least once")
	}
	if traversal.calls != 1 {
		t.Fatalf("expected traversal to run once, got %d", traversal.calls)
	}
}

type tickingClock struct{}

func (*tickingClock) Now() time.Time { return time.Time{} }

func (*tickingClock) Sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("ticking clock: %w", ctx.Err())
	}
}
