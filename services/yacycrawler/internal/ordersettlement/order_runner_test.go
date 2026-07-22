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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/ordersettlement"
)

type fakeTraversal struct {
	err   error
	calls int
	block <-chan struct{}
}

func (f *fakeTraversal) Traverse(ctx context.Context, _ crawlcapability.DeliveredOrder) error {
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

func (*recordingObserver) PageFetched()                {}
func (*recordingObserver) PublicationWaited()          {}
func (*recordingObserver) FetchObserved(time.Duration) {}
func (*recordingObserver) PageDisposed(string)         {}
func (*recordingObserver) PagePublished(string)        {}
func (*recordingObserver) RefusalHonored(string)       {}
func (*recordingObserver) BudgetExhausted()            {}

type manualClock struct{}

func (manualClock) Now() time.Time { return time.Time{} }

func (manualClock) Sleep(ctx context.Context, _ time.Duration) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("manual clock: %w", err)
	}
	return nil
}

type settlement struct {
	acked   int
	retried int
	ackErr  error
}

func deliver(
	t *testing.T,
	runner *ordersettlement.OrderRunner,
	s *settlement,
	extendOwnership func(context.Context) error,
) {
	t.Helper()
	deliveries := make(chan crawlcapability.DeliveredOrder, 1)
	deliveries <- crawlcapability.DeliveredOrder{
		Order:           yacycrawlcontract.CrawlOrder{OrderID: "o1"},
		Ack:             func(context.Context) error { s.acked++; return s.ackErr },
		Retry:           func(context.Context) error { s.retried++; return nil },
		ExtendOwnership: extendOwnership,
	}
	close(deliveries)
	if err := runner.Run(context.Background(), deliveries); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func noopExtend(context.Context) error { return nil }

func TestRunAcksAndCompletesOnSuccess(t *testing.T) {
	observer := &recordingObserver{}
	runner := ordersettlement.NewOrderRunner(observer, &fakeTraversal{}, manualClock{}, 0)
	s := &settlement{}

	deliver(t, runner, s, noopExtend)
	if s.acked != 1 || observer.completed != 1 || observer.received != 1 {
		t.Fatalf("expected ack+complete: %+v observer=%+v", s, observer)
	}
}

func TestRunRedeliversOnTraversalError(t *testing.T) {
	observer := &recordingObserver{}
	runner := ordersettlement.NewOrderRunner(
		observer, &fakeTraversal{err: errors.New("boom")}, manualClock{}, 0,
	)
	s := &settlement{}

	deliver(t, runner, s, noopExtend)
	if s.retried != 1 || s.acked != 0 || observer.redelivered != 1 || observer.completed != 0 {
		t.Fatalf("expected redelivery: %+v observer=%+v", s, observer)
	}
}

func TestRunToleratesAckError(t *testing.T) {
	observer := &recordingObserver{}
	runner := ordersettlement.NewOrderRunner(observer, &fakeTraversal{}, manualClock{}, 0)
	s := &settlement{ackErr: errors.New("ack failed")}

	deliver(t, runner, s, noopExtend)
	if observer.completed != 0 {
		t.Fatalf("ack failure should not count as completed: %+v", observer)
	}
}

func TestRunStopsWhenDeliveriesClose(t *testing.T) {
	runner := ordersettlement.NewOrderRunner(
		&recordingObserver{},
		&fakeTraversal{},
		manualClock{},
		0,
	)
	deliveries := make(chan crawlcapability.DeliveredOrder)
	close(deliveries)

	if err := runner.Run(context.Background(), deliveries); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRunStopsWhenContextCancelled(t *testing.T) {
	runner := ordersettlement.NewOrderRunner(
		&recordingObserver{},
		&fakeTraversal{},
		manualClock{},
		0,
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := runner.Run(ctx, make(chan crawlcapability.DeliveredOrder)); err == nil {
		t.Fatal("cancelled context should end the run with an error")
	}
}

func TestRunRenewsOwnershipWhileTraversing(t *testing.T) {
	gate := make(chan struct{})
	traversal := &fakeTraversal{block: gate}
	runner := ordersettlement.NewOrderRunner(
		&recordingObserver{}, traversal, &tickingClock{}, time.Millisecond,
	)

	var openOnce sync.Once
	var renewed atomic.Int64
	extend := func(context.Context) error {
		renewed.Add(1)
		openOnce.Do(func() { close(gate) })
		return nil
	}

	s := &settlement{}
	deliver(t, runner, s, extend)

	if renewed.Load() == 0 {
		t.Fatal("expected ownership heartbeat to extend at least once")
	}
	if traversal.calls != 1 {
		t.Fatalf("expected traversal to run once, got %d", traversal.calls)
	}
}

type tickingClock struct{}

func (*tickingClock) Now() time.Time { return time.Time{} }

func (*tickingClock) Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("ticking clock: %w", ctx.Err())
	}
}
