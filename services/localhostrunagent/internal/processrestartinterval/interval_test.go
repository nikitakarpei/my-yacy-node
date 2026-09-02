package processrestartinterval_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/localhostrunagent/internal/processrestartinterval"
)

const shortestInterval = 200 * time.Millisecond

func TestTheExitIsHeldUntilTheShortestIntervalHasPassed(t *testing.T) {
	processStart := time.Now()

	processrestartinterval.HoldTheExit(t.Context(), processStart, shortestInterval)

	if held := time.Since(processStart); held < shortestInterval {
		t.Errorf("the exit was held for %v, want at least %v", held, shortestInterval)
	}
}

func TestTheExitOfAProcessThatRanLongerIsNotHeld(t *testing.T) {
	calledAt := time.Now()

	processrestartinterval.HoldTheExit(
		t.Context(), calledAt.Add(-time.Hour), shortestInterval,
	)

	if held := time.Since(calledAt); held >= shortestInterval {
		t.Errorf("the exit was held for %v, want no hold", held)
	}
}

func TestTheExitIsNotHeldAfterTheProcessIsSignalled(t *testing.T) {
	signalled, signal := context.WithCancel(t.Context())
	signal()

	calledAt := time.Now()

	processrestartinterval.HoldTheExit(signalled, calledAt, shortestInterval)

	if held := time.Since(calledAt); held >= shortestInterval {
		t.Errorf("the exit was held for %v, want no hold", held)
	}
}
