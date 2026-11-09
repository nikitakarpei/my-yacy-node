// Package fetchingduty decides who fetches pages: the page services of the peer itself, or its
// paired companions. The peer fetches while no device is paired or companions do not fetch
// pages, and while no companion is connected if the peer fills in for them. Otherwise it holds
// its intake back. The package also tells when nobody fetches because no companion is connected.
package fetchingduty

import (
	"context"
	"log/slog"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
)

type Duty struct {
	released           chan struct{}
	fetching           companionsettings.Fetching
	devicePaired       bool
	companionConnected bool
	mu                 sync.Mutex
}

func New() *Duty {
	released := make(chan struct{})
	close(released)
	return &Duty{released: released}
}

func (d *Duty) FetchingChanged(ctx context.Context, fetching companionsettings.Fetching) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.fetching = fetching
	d.settle(ctx)
}

func (d *Duty) FirstDevicePaired(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.devicePaired = true
	d.settle(ctx)
}

func (d *Duty) LastDeviceUnpaired(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.devicePaired = false
	d.settle(ctx)
}

func (d *Duty) FirstCompanionConnected(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.companionConnected = true
	d.settle(ctx)
}

func (d *Duty) LastCompanionDisconnected(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.companionConnected = false
	d.settle(ctx)
}

func (d *Duty) settle(ctx context.Context) {
	admits := d.peerTakesWork()
	if admits == isClosed(d.released) {
		return
	}
	if admits {
		close(d.released)
		slog.DebugContext(ctx, "page services of the peer take work")
		return
	}
	d.released = make(chan struct{})
	slog.DebugContext(ctx, "page services of the peer hold their intake for the companions")
}

func (d *Duty) peerTakesWork() bool {
	return !d.devicePaired || !d.companionsCover()
}

func (d *Duty) companionsCover() bool {
	return d.fetching.CompanionsFetch && (!d.fetching.PeerFillsIn || d.companionConnected)
}

func isClosed(released chan struct{}) bool {
	select {
	case <-released:
		return true
	default:
		return false
	}
}

func (d *Duty) AwaitAdmission(ctx context.Context) error {
	d.mu.Lock()
	released := d.released
	d.mu.Unlock()
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

func (d *Duty) FetchingPaused() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return !d.peerTakesWork() && !d.companionConnected
}
