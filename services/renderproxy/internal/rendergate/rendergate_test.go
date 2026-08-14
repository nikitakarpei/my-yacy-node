package rendergate_test

import (
	"context"
	"errors"
	"fmt"
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

type stubMetrics struct {
	waited    atomic.Int64
	succeeded atomic.Int64
	failed    atomic.Int64
}

func (m *stubMetrics) RenderWaited()                { m.waited.Add(1) }
func (m *stubMetrics) RenderSucceeded()             { m.succeeded.Add(1) }
func (m *stubMetrics) RenderFailed(reason string)   { m.failed.Add(1) }
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
