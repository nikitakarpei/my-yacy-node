package background_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderplacers/background"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type recordingCrawlOrderPlacementObserver struct {
	mu       sync.Mutex
	accepted int
	refused  int
}

func (o *recordingCrawlOrderPlacementObserver) CrawlOrderPlacementAccepted(
	context.Context,
	string,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.accepted++
}

func (o *recordingCrawlOrderPlacementObserver) CrawlOrderPlacementRefused(
	context.Context,
	string,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.refused++
}

func (o *recordingCrawlOrderPlacementObserver) acceptedCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.accepted
}

func (o *recordingCrawlOrderPlacementObserver) refusedCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.refused
}

type recordingCrawlOrderPlacer struct {
	mu       sync.Mutex
	orders   []string
	deadline bool
	canceled bool
}

func (p *recordingCrawlOrderPlacer) Place(
	ctx context.Context,
	order yacycrawlcontract.CrawlOrder,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.orders = append(p.orders, order.OrderID)
	_, p.deadline = ctx.Deadline()
	p.canceled = ctx.Err() != nil
}

func (p *recordingCrawlOrderPlacer) placedOrders() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.orders...)
}

type blockingCrawlOrderPlacer struct {
	started sync.WaitGroup
	release <-chan struct{}
}

func (p *blockingCrawlOrderPlacer) Place(context.Context, yacycrawlcontract.CrawlOrder) {
	p.started.Done()
	<-p.release
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}

func TestPlaceAcceptsOrderAndCarriesItToThePlacer(t *testing.T) {
	observer := &recordingCrawlOrderPlacementObserver{}
	carrier := &recordingCrawlOrderPlacer{}
	placer := background.New(carrier, observer, time.Second, 1)

	placer.Place(context.Background(), yacycrawlcontract.CrawlOrder{OrderID: "o1"})

	waitFor(t, func() bool { return len(carrier.placedOrders()) == 1 })
	if observer.acceptedCount() != 1 || observer.refusedCount() != 0 {
		t.Fatalf(
			"accepted = %d, refused = %d, want 1 and 0",
			observer.acceptedCount(), observer.refusedCount(),
		)
	}
}

func TestPlaceOutlivesTheCallerUnderItsOwnTimeBound(t *testing.T) {
	observer := &recordingCrawlOrderPlacementObserver{}
	carrier := &recordingCrawlOrderPlacer{}
	placer := background.New(carrier, observer, time.Second, 1)

	ctx, cancel := context.WithCancel(context.Background())
	placer.Place(ctx, yacycrawlcontract.CrawlOrder{OrderID: "o1"})
	cancel()

	waitFor(t, func() bool { return len(carrier.placedOrders()) == 1 })
	carrier.mu.Lock()
	defer carrier.mu.Unlock()
	if carrier.canceled {
		t.Fatal("placement should not be canceled with the caller")
	}
	if !carrier.deadline {
		t.Fatal("placement should carry a deadline of its own")
	}
}

func TestPlaceRefusesOrderWhenSaturated(t *testing.T) {
	release := make(chan struct{})
	observer := &recordingCrawlOrderPlacementObserver{}
	blockingPlacer := &blockingCrawlOrderPlacer{release: release}
	blockingPlacer.started.Add(1)
	placer := background.New(blockingPlacer, observer, time.Second, 1)

	placer.Place(context.Background(), yacycrawlcontract.CrawlOrder{OrderID: "o1"})
	blockingPlacer.started.Wait()
	placer.Place(context.Background(), yacycrawlcontract.CrawlOrder{OrderID: "o2"})

	if refused := observer.refusedCount(); refused != 1 {
		t.Fatalf("refused = %d, want 1", refused)
	}
	if accepted := observer.acceptedCount(); accepted != 1 {
		t.Fatalf("accepted = %d, want 1", accepted)
	}

	close(release)
}
