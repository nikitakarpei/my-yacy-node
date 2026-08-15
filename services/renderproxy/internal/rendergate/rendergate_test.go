package rendergate_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendergate"
)

type stubRenderer struct {
	page  renderedpage.Page
	err   error
	delay time.Duration
}

func (s *stubRenderer) Render(ctx context.Context, targetURL string) (renderedpage.Page, error) {
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return renderedpage.Page{}, fmt.Errorf("stub render canceled: %w", ctx.Err())
	}

	return s.page, s.err
}

type blockingRenderer struct {
	entered chan struct{}
	release chan struct{}
}

func (r *blockingRenderer) Render(context.Context, string) (renderedpage.Page, error) {
	close(r.entered)
	<-r.release

	return renderedpage.Page{}, nil
}

type stubMetrics struct {
	waited       atomic.Int64
	succeeded    atomic.Int64
	failed       atomic.Int64
	failedReason atomic.Pointer[string]
}

func (m *stubMetrics) RenderWaited()    { m.waited.Add(1) }
func (m *stubMetrics) RenderSucceeded() { m.succeeded.Add(1) }

func (m *stubMetrics) RenderFailed(reason string) {
	m.failed.Add(1)
	m.failedReason.Store(&reason)
}

func (m *stubMetrics) RenderObserved(time.Duration) {}

func TestRenderFailsWhenPageTooLarge(t *testing.T) {
	inner := &stubRenderer{page: renderedpage.Page{Body: make([]byte, 100)}}
	metrics := &stubMetrics{}
	gated := rendergate.New(inner, 1, time.Second, 10, metrics)

	if _, err := gated.Render(context.Background(), "http://example.com"); err == nil {
		t.Fatal("expected error for oversized page")
	}
	if metrics.failed.Load() != 1 {
		t.Fatalf("failed count = %d, want 1", metrics.failed.Load())
	}
}

func TestRenderAppliesDeadline(t *testing.T) {
	inner := &stubRenderer{delay: 50 * time.Millisecond}
	metrics := &stubMetrics{}
	gated := rendergate.New(inner, 1, 5*time.Millisecond, 1024, metrics)

	_, err := gated.Render(context.Background(), "http://example.com")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestRenderPropagatesInnerError(t *testing.T) {
	inner := &stubRenderer{err: errors.New("boom")}
	metrics := &stubMetrics{}
	gated := rendergate.New(inner, 1, time.Second, 1024, metrics)

	if _, err := gated.Render(context.Background(), "http://example.com"); err == nil {
		t.Fatal("expected error")
	}
	if metrics.succeeded.Load() != 0 {
		t.Fatal("did not expect success recorded")
	}
}

func TestRenderWaitsForSlotWhenConcurrencyCapReached(t *testing.T) {
	gated, metrics, release, held := gateWithSlotHeld(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = gated.Render(ctx, "http://example.com")

	close(release)
	held.Wait()

	if metrics.waited.Load() != 1 {
		t.Fatalf("waited count = %d, want 1", metrics.waited.Load())
	}
}

func gateWithSlotHeld(
	t *testing.T,
) (*rendergate.Renderer, *stubMetrics, chan<- struct{}, *sync.WaitGroup) {
	t.Helper()

	metrics := &stubMetrics{}
	inner := &blockingRenderer{entered: make(chan struct{}), release: make(chan struct{})}
	gated := rendergate.New(inner, 1, time.Second, 1024, metrics)

	var held sync.WaitGroup
	held.Go(func() {
		_, _ = gated.Render(context.Background(), "http://example.com")
	})
	<-inner.entered

	return gated, metrics, inner.release, &held
}

func TestRenderReportsSlotWaitTimeout(t *testing.T) {
	gated, metrics, release, held := gateWithSlotHeld(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := gated.Render(ctx, "http://example.com"); err == nil {
		t.Fatal("expected error from render with an already cancelled context")
	}

	close(release)
	held.Wait()

	reason := metrics.failedReason.Load()
	if reason == nil || *reason != rendergate.ReasonSlotWaitTimeout {
		t.Fatalf("failure reason = %v, want %s", reason, rendergate.ReasonSlotWaitTimeout)
	}
}
