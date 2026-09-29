// Package readtimer cancels reading an answer that passes its deadline or
// waits too long for its next bytes.
package readtimer

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

type Timer struct {
	clock      Clock
	cancel     context.CancelFunc
	arming     sync.Mutex
	stop       func()
	wasExpired atomic.Bool
}

func New(clock Clock, cancel context.CancelFunc) *Timer {
	return &Timer{clock: clock, cancel: cancel, stop: func() {}}
}

func (t *Timer) EndReadingAt(deadline time.Time) {
	t.expireAfter(deadline.Sub(t.clock.Now()))
}

func (t *Timer) IdleLimitedFrom(answerBody io.Reader, idleTimeout time.Duration) io.Reader {
	return idleLimitedReader{timer: t, answerBody: answerBody, idleTimeout: idleTimeout}
}

func (t *Timer) Expired() bool {
	return t.wasExpired.Load()
}

func (t *Timer) Stop() {
	t.arming.Lock()
	defer t.arming.Unlock()
	t.stop()
}

func (t *Timer) expireAfter(timeout time.Duration) {
	t.arming.Lock()
	defer t.arming.Unlock()
	t.stop()
	t.stop = t.clock.After(timeout, t.expire)
}

func (t *Timer) expire() {
	t.wasExpired.Store(true)
	t.cancel()
}

type idleLimitedReader struct {
	timer       *Timer
	answerBody  io.Reader
	idleTimeout time.Duration
}

func (r idleLimitedReader) Read(chunk []byte) (int, error) {
	r.timer.expireAfter(r.idleTimeout)
	defer r.timer.Stop()
	//nolint:wrapcheck // io.EOF reaches the caller of a reader unwrapped
	return r.answerBody.Read(chunk)
}
