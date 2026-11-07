package companionpresence

import (
	"context"
	"log/slog"
	"sync"
)

type IntakeHold struct {
	released chan struct{}
	mu       sync.Mutex
}

func NewIntakeHold() *IntakeHold {
	released := make(chan struct{})
	close(released)
	return &IntakeHold{released: released}
}

func (h *IntakeHold) CompanionsConnected(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.released = make(chan struct{})
	slog.DebugContext(ctx, "page services of the peer hold their intake for the companions")
}

func (h *IntakeHold) CompanionsLeft(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	close(h.released)
	slog.DebugContext(ctx, "page services of the peer take work again")
}

func (h *IntakeHold) AwaitAdmission(ctx context.Context) error {
	h.mu.Lock()
	released := h.released
	h.mu.Unlock()
	select {
	case <-released:
		return nil
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck // the caller names the wait.
	}
}
