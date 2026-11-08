// Package peerintake decides when the page services of the peer itself take work: always while
// no device is paired or companions do not fetch pages, and while no companion is connected if
// the peer fills in for them. Otherwise it holds their intake back, so only companions fetch
// pages.
package peerintake

import (
	"context"
	"log/slog"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
)

type Admission struct {
	released           chan struct{}
	choice             companionsettings.Choice
	devicePaired       bool
	companionConnected bool
	mu                 sync.Mutex
}

func New() *Admission {
	released := make(chan struct{})
	close(released)
	return &Admission{released: released}
}

func (a *Admission) ChoiceMade(ctx context.Context, choice companionsettings.Choice) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.choice = choice
	a.settle(ctx)
}

func (a *Admission) FirstDevicePaired(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.devicePaired = true
	a.settle(ctx)
}

func (a *Admission) LastDeviceUnpaired(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.devicePaired = false
	a.settle(ctx)
}

func (a *Admission) FirstCompanionConnected(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.companionConnected = true
	a.settle(ctx)
}

func (a *Admission) LastCompanionDisconnected(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.companionConnected = false
	a.settle(ctx)
}

func (a *Admission) settle(ctx context.Context) {
	companionsCover := a.choice.OnlyCompanionsFetch() ||
		a.choice.CompanionsFetch && a.companionConnected
	admits := !a.devicePaired || !companionsCover
	if admits == isClosed(a.released) {
		return
	}
	if admits {
		close(a.released)
		slog.DebugContext(ctx, "page services of the peer take work")
		return
	}
	a.released = make(chan struct{})
	slog.DebugContext(ctx, "page services of the peer hold their intake for the companions")
}

func isClosed(released chan struct{}) bool {
	select {
	case <-released:
		return true
	default:
		return false
	}
}

func (a *Admission) AwaitAdmission(ctx context.Context) error {
	a.mu.Lock()
	released := a.released
	a.mu.Unlock()
	if isClosed(released) {
		return nil
	}
	select {
	case <-released:
		return nil
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck // the caller names the wait.
	}
}
