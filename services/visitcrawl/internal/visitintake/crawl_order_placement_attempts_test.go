package visitintake_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/visitintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type recordingCrawlOrderPlacementObserver struct {
	mu       sync.Mutex
	placed   int
	unplaced int
}

func (o *recordingCrawlOrderPlacementObserver) CrawlOrderPlaced(context.Context, string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.placed++
}

func (o *recordingCrawlOrderPlacementObserver) CrawlOrderPlacementFailed(
	context.Context,
	string,
	error,
) {
	o.recordUnplaced()
}

func (o *recordingCrawlOrderPlacementObserver) CrawlOrderPlacementSkippedBecauseSaturated(
	context.Context,
	string,
) {
	o.recordUnplaced()
}

func (o *recordingCrawlOrderPlacementObserver) recordUnplaced() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.unplaced++
}

func (o *recordingCrawlOrderPlacementObserver) placedCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.placed
}

func (o *recordingCrawlOrderPlacementObserver) unplacedCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.unplaced
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

func TestCrawlOrderPlacementAttemptsRecordsSuccess(t *testing.T) {
	observer := &recordingCrawlOrderPlacementObserver{}
	placementAttempts := visitintake.NewCrawlOrderPlacementAttempts(
		func(_ context.Context, _ yacycrawlcontract.CrawlOrder) error { return nil },
		observer, time.Second, 1,
	)
	placementAttempts.Start(yacycrawlcontract.CrawlOrder{OrderID: "o1"})
	waitFor(t, func() bool { return observer.placedCount() == 1 })
	if unplaced := observer.unplacedCount(); unplaced != 0 {
		t.Fatalf("unplaced = %d, want 0", unplaced)
	}
}

func TestCrawlOrderPlacementAttemptsRecordsFailure(t *testing.T) {
	observer := &recordingCrawlOrderPlacementObserver{}
	placementAttempts := visitintake.NewCrawlOrderPlacementAttempts(
		func(_ context.Context, _ yacycrawlcontract.CrawlOrder) error { return errors.New("broker down") },
		observer,
		time.Second,
		1,
	)
	placementAttempts.Start(yacycrawlcontract.CrawlOrder{OrderID: "o1"})
	waitFor(t, func() bool { return observer.unplacedCount() == 1 })
}

func TestCrawlOrderPlacementAttemptsSaturationSkipsWithoutBlocking(t *testing.T) {
	release := make(chan struct{})
	var started sync.WaitGroup
	started.Add(1)

	observer := &recordingCrawlOrderPlacementObserver{}
	placementAttempts := visitintake.NewCrawlOrderPlacementAttempts(
		func(_ context.Context, _ yacycrawlcontract.CrawlOrder) error {
			started.Done()
			<-release
			return nil
		},
		observer, time.Second, 1,
	)

	placementAttempts.Start(yacycrawlcontract.CrawlOrder{OrderID: "o1"})
	started.Wait()

	done := make(chan struct{})
	go func() {
		placementAttempts.Start(yacycrawlcontract.CrawlOrder{OrderID: "o2"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("second attempt blocked instead of skipping")
	}

	waitFor(t, func() bool { return observer.unplacedCount() == 1 })
	close(release)
	waitFor(t, func() bool { return observer.placedCount() == 1 })
}
