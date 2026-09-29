// Package readingcancel cancels reading an upstream response that passes its
// deadline or waits too long for its next bytes.
package readingcancel

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

type Clock interface {
	Now() time.Time
	After(timeout time.Duration, expire func()) (stop func())
}

type Canceller struct {
	clock         Clock
	cancelReading context.CancelFunc
	arming        sync.Mutex
	stop          func()
	wasCancelled  atomic.Bool
}

func New(clock Clock, cancelReading context.CancelFunc) *Canceller {
	return &Canceller{clock: clock, cancelReading: cancelReading, stop: func() {}}
}

func (c *Canceller) CancelAt(deadline time.Time) {
	c.cancelAfter(deadline.Sub(c.clock.Now()))
}

func (c *Canceller) IdleLimitedFrom(
	upstreamResponseBody io.Reader,
	idleTimeout time.Duration,
) io.Reader {
	return idleLimitedReader{
		canceller:            c,
		upstreamResponseBody: upstreamResponseBody,
		idleTimeout:          idleTimeout,
	}
}

func (c *Canceller) Cancelled() bool {
	return c.wasCancelled.Load()
}

func (c *Canceller) Stop() {
	c.arming.Lock()
	defer c.arming.Unlock()
	c.stop()
}

func (c *Canceller) cancelAfter(timeout time.Duration) {
	c.arming.Lock()
	defer c.arming.Unlock()
	c.stop()
	c.stop = c.clock.After(timeout, c.cancel)
}

func (c *Canceller) cancel() {
	c.wasCancelled.Store(true)
	c.cancelReading()
}

type idleLimitedReader struct {
	canceller            *Canceller
	upstreamResponseBody io.Reader
	idleTimeout          time.Duration
}

func (r idleLimitedReader) Read(chunk []byte) (int, error) {
	r.canceller.cancelAfter(r.idleTimeout)
	defer r.canceller.Stop()
	//nolint:wrapcheck // io.EOF reaches the caller of a reader unwrapped
	return r.upstreamResponseBody.Read(chunk)
}
