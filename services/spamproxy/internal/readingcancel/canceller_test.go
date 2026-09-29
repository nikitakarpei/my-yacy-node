package readingcancel_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/readingcancel"
)

var now = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

type armedTimer struct {
	timeout time.Duration
	expire  func()
	stopped bool
}

type fakeClock struct {
	armedTimers []*armedTimer
}

func (c *fakeClock) Now() time.Time { return now }

func (c *fakeClock) After(timeout time.Duration, expire func()) func() {
	armed := &armedTimer{timeout: timeout, expire: expire}
	c.armedTimers = append(c.armedTimers, armed)
	return func() { armed.stopped = true }
}

type cancelRecord struct {
	cancellations int
}

func (r *cancelRecord) cancel() { r.cancellations++ }

func TestReadingPastTheDeadlineCancelsTheUpstreamResponse(t *testing.T) {
	clock := &fakeClock{}
	cancels := &cancelRecord{}
	canceller := readingcancel.New(clock, cancels.cancel)

	canceller.CancelAt(now.Add(3 * time.Second))
	clock.armedTimers[0].expire()

	if clock.armedTimers[0].timeout != 3*time.Second || cancels.cancellations != 1 ||
		!canceller.Cancelled() {
		t.Fatalf("timeout %v, cancellations %d, cancelled %v",
			clock.armedTimers[0].timeout, cancels.cancellations, canceller.Cancelled())
	}
}

func TestAStoppedCancellerHasNotCancelled(t *testing.T) {
	clock := &fakeClock{}
	canceller := readingcancel.New(clock, (&cancelRecord{}).cancel)

	canceller.CancelAt(now.Add(time.Second))
	canceller.Stop()

	if !clock.armedTimers[0].stopped || canceller.Cancelled() {
		t.Fatalf("stopped %v, cancelled %v", clock.armedTimers[0].stopped, canceller.Cancelled())
	}
}
