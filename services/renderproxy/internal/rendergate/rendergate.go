// Package rendergate bounds concurrency and deadline around a renderer and reports
// how each render ended.
package rendergate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

const (
	ReasonRenderFailed    = "render_failed"
	ReasonTooLarge        = "too_large"
	ReasonSlotWaitTimeout = "slot_wait_timeout"
)

type Metrics interface {
	RenderWaited()
	RenderSucceeded()
	RenderFailed(reason string)
	RenderObserved(elapsed time.Duration)
}

type Renderer struct {
	inner    renderedpage.Renderer
	slots    chan struct{}
	deadline time.Duration
	metrics  Metrics
}

func New(
	inner renderedpage.Renderer,
	concurrency int,
	deadline time.Duration,
	metrics Metrics,
) *Renderer {
	return &Renderer{
		inner:    inner,
		slots:    make(chan struct{}, concurrency),
		deadline: deadline,
		metrics:  metrics,
	}
}

func (r *Renderer) Render(ctx context.Context, targetURL string) (renderedpage.Page, error) {
	if err := r.acquire(ctx); err != nil {
		return renderedpage.Page{}, err
	}
	defer func() { <-r.slots }()

	renderCtx, cancel := context.WithTimeout(ctx, r.deadline)
	defer cancel()

	start := time.Now()
	page, err := r.inner.Render(renderCtx, targetURL)
	r.metrics.RenderObserved(time.Since(start))
	if err != nil {
		r.metrics.RenderFailed(failureReasonOf(err))
		return renderedpage.Page{}, fmt.Errorf("render %s: %w", targetURL, err)
	}

	r.metrics.RenderSucceeded()
	return page, nil
}

func failureReasonOf(err error) string {
	if errors.Is(err, renderedpage.ErrTooLarge) {
		return ReasonTooLarge
	}
	return ReasonRenderFailed
}

func (r *Renderer) acquire(ctx context.Context) error {
	select {
	case r.slots <- struct{}{}:
		return nil
	default:
	}

	r.metrics.RenderWaited()
	select {
	case r.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		r.metrics.RenderFailed(ReasonSlotWaitTimeout)
		return fmt.Errorf("wait for render slot: %w", ctx.Err())
	}
}
