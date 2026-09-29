package readingcancel_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/readingcancel"
)

func TestEachReadWaitsAtMostTheIdleTimeoutAndOnlyWhileItReads(t *testing.T) {
	clock := &fakeClock{}
	canceller := readingcancel.New(clock, (&cancelRecord{}).cancel)
	canceller.CancelAt(now.Add(time.Second))

	body, err := io.ReadAll(readingcancel.IdleCancellingReaderFrom(
		strings.NewReader("page"),
		canceller,
		30*time.Second,
	))
	if err != nil || string(body) != "page" {
		t.Fatalf("body %q, error %v", body, err)
	}
	for position, armed := range clock.armedTimers {
		if !armed.stopped || (position > 0 && armed.timeout != 30*time.Second) {
			t.Fatalf("timer %d: timeout %v, stopped %v", position, armed.timeout, armed.stopped)
		}
	}
	if len(clock.armedTimers) < 3 {
		t.Fatalf("armed %d timers, want one per read after the deadline", len(clock.armedTimers))
	}
}
