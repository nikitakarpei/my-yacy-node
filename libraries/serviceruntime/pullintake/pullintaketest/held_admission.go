package pullintaketest

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type HeldAdmission struct {
	asked    chan struct{}
	released <-chan struct{}
}

func AdmissionHeldUntil(release <-chan struct{}) HeldAdmission {
	return HeldAdmission{asked: make(chan struct{}, 1), released: release}
}

func (a HeldAdmission) AwaitAdmission(ctx context.Context) error {
	select {
	case a.asked <- struct{}{}:
	default:
	}
	select {
	case <-a.released:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("await admission: %w", ctx.Err())
	}
}

func (a HeldAdmission) AwaitAsked(t *testing.T, within time.Duration) {
	t.Helper()
	select {
	case <-a.asked:
	case <-time.After(within):
		t.Fatal("the service never asked the intake admission")
	}
}
