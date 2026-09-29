package readingcancel_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/readingcancel"
)

type stallingReader struct {
	clock *fakeClock
}

func (r stallingReader) Read([]byte) (int, error) {
	r.clock.armedTimers[len(r.clock.armedTimers)-1].fire()
	return 0, io.ErrUnexpectedEOF
}

func TestEachReadWaitsAtMostTheIdleTimeoutAndOnlyWhileItReads(t *testing.T) {
	clock := &fakeClock{}
	readingCtx, cancelReading := context.WithCancelCause(t.Context())

	body, err := io.ReadAll(readingcancel.IdleCancellingReaderFrom(
		strings.NewReader("page"),
		clock,
		cancelReading,
		30*time.Second,
	))
	if err != nil || string(body) != "page" || readingCtx.Err() != nil {
		t.Fatalf("body %q, error %v, cause %v", body, err, context.Cause(readingCtx))
	}
	for position, armed := range clock.armedTimers {
		if !armed.stopped || armed.timeout != 30*time.Second {
			t.Fatalf("timer %d: timeout %v, stopped %v", position, armed.timeout, armed.stopped)
		}
	}
	if len(clock.armedTimers) < 2 {
		t.Fatalf("armed %d timers, want one per read", len(clock.armedTimers))
	}
}

func TestAReadThatStallsPastTheIdleTimeoutIsCancelledForTheIdleTimeout(t *testing.T) {
	clock := &fakeClock{}
	readingCtx, cancelReading := context.WithCancelCause(t.Context())

	_, _ = io.ReadAll(readingcancel.IdleCancellingReaderFrom(
		stallingReader{clock: clock},
		clock,
		cancelReading,
		30*time.Second,
	))

	if cause := context.Cause(readingCtx); !errors.Is(cause, readingcancel.ErrIdleTimeoutPassed) {
		t.Fatalf("cause %v", cause)
	}
}
