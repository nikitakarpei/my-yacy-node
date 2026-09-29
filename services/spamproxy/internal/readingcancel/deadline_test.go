package readingcancel_test

import (
	"context"
	"errors"
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

func (t *armedTimer) fire() {
	if !t.stopped {
		t.expire()
	}
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

func TestReadingPastTheDeadlineIsCancelledForTheDeadline(t *testing.T) {
	clock := &fakeClock{}
	readingCtx, cancelReading := context.WithCancelCause(t.Context())

	readingcancel.CancelAt(clock, now.Add(3*time.Second), cancelReading)
	clock.armedTimers[0].fire()

	cause := context.Cause(readingCtx)
	if clock.armedTimers[0].timeout != 3*time.Second ||
		!errors.Is(cause, readingcancel.ErrDeadlinePassed) {
		t.Fatalf("timeout %v, cause %v", clock.armedTimers[0].timeout, cause)
	}
}

func TestReadingIsNotCancelledAfterTheDeadlineStops(t *testing.T) {
	clock := &fakeClock{}
	readingCtx, cancelReading := context.WithCancelCause(t.Context())

	stop := readingcancel.CancelAt(clock, now.Add(time.Second), cancelReading)
	stop()
	clock.armedTimers[0].fire()

	if readingCtx.Err() != nil {
		t.Fatalf("cause %v", context.Cause(readingCtx))
	}
}
