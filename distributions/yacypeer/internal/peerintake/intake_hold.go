// Package peerintake decides when the page services of the peer itself take work: always while
// no device is paired or companions do not fetch pages, and while no companion is connected if
// the peer fills in for them. Otherwise it holds their intake back, so only companions fetch
// pages.
package peerintake

import (
	"context"
	"log/slog"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/fetchsettings"
)

type IntakeHold struct {
	released           chan struct{}
	choice             fetchsettings.Choice
	devicePaired       bool
	companionConnected bool
	mu                 sync.Mutex
}

func NewIntakeHold() *IntakeHold {
	released := make(chan struct{})
	close(released)
	return &IntakeHold{released: released}
}

func (h *IntakeHold) ChoiceMade(ctx context.Context, choice fetchsettings.Choice) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.choice = choice
	h.settle(ctx)
}

func (h *IntakeHold) FirstDevicePaired(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.devicePaired = true
	h.settle(ctx)
}

func (h *IntakeHold) LastDeviceUnpaired(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.devicePaired = false
	h.settle(ctx)
}

func (h *IntakeHold) CompanionsConnected(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.companionConnected = true
	h.settle(ctx)
}

func (h *IntakeHold) CompanionsLeft(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.companionConnected = false
	h.settle(ctx)
}

func (h *IntakeHold) settle(ctx context.Context) {
	admits := !h.devicePaired || !h.choice.CompanionsFetch ||
		(h.choice.PeerFillsIn && !h.companionConnected)
	if admits == isClosed(h.released) {
		return
	}
	if admits {
		close(h.released)
		slog.DebugContext(ctx, "page services of the peer take work")
		return
	}
	h.released = make(chan struct{})
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

func (h *IntakeHold) AwaitAdmission(ctx context.Context) error {
	h.mu.Lock()
	released := h.released
	h.mu.Unlock()
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
