// Package rendergate bounds renderer concurrency and duration.
package rendergate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

type RenderObserver interface {
	RenderSucceeded(ctx context.Context, targetURL string, renderDuration time.Duration)
	RenderTimedOut(ctx context.Context, targetURL string, renderDuration time.Duration, cause error)
	RenderCallerGaveUp(
		ctx context.Context,
		targetURL string,
		renderDuration time.Duration,
		cause error,
	)
	RenderPageTooLarge(
		ctx context.Context,
		targetURL string,
		renderDuration time.Duration,
		cause error,
	)
	RenderFailed(ctx context.Context, targetURL string, renderDuration time.Duration, cause error)
}

type DeadlineRenderer struct {
	inner          renderedpage.Renderer
	renderDeadline time.Duration
	renderObserver RenderObserver
}

func NewDeadlineRenderer(
	inner renderedpage.Renderer,
	renderDeadline time.Duration,
	renderObserver RenderObserver,
) *DeadlineRenderer {
	return &DeadlineRenderer{
		inner:          inner,
		renderDeadline: renderDeadline,
		renderObserver: renderObserver,
	}
}

func (r *DeadlineRenderer) Render(
	ctx context.Context,
	target renderedpage.Target,
) (renderedpage.Page, error) {
	renderContext, cancel := context.WithTimeout(ctx, r.renderDeadline)
	defer cancel()

	renderStarted := time.Now()
	page, err := r.inner.Render(renderContext, target)
	renderDuration := time.Since(renderStarted)
	if err != nil {
		r.reportRenderFailure(ctx, target.URL, renderDuration, err)
		return renderedpage.Page{}, fmt.Errorf("render %s: %w", target.URL, err)
	}
	r.renderObserver.RenderSucceeded(ctx, target.URL, renderDuration)
	return page, nil
}

func (r *DeadlineRenderer) reportRenderFailure(
	ctx context.Context,
	targetURL string,
	renderDuration time.Duration,
	cause error,
) {
	switch {
	case ctx.Err() != nil:
		r.renderObserver.RenderCallerGaveUp(ctx, targetURL, renderDuration, cause)
	case errors.Is(cause, context.DeadlineExceeded):
		r.renderObserver.RenderTimedOut(ctx, targetURL, renderDuration, cause)
	case errors.Is(cause, renderedpage.ErrTooLarge):
		r.renderObserver.RenderPageTooLarge(ctx, targetURL, renderDuration, cause)
	default:
		r.renderObserver.RenderFailed(ctx, targetURL, renderDuration, cause)
	}
}

type RenderObservers []RenderObserver

func (observers RenderObservers) RenderSucceeded(
	ctx context.Context,
	targetURL string,
	renderDuration time.Duration,
) {
	for _, observer := range observers {
		observer.RenderSucceeded(ctx, targetURL, renderDuration)
	}
}

func (observers RenderObservers) RenderTimedOut(
	ctx context.Context,
	targetURL string,
	renderDuration time.Duration,
	cause error,
) {
	for _, observer := range observers {
		observer.RenderTimedOut(ctx, targetURL, renderDuration, cause)
	}
}

func (observers RenderObservers) RenderCallerGaveUp(
	ctx context.Context,
	targetURL string,
	renderDuration time.Duration,
	cause error,
) {
	for _, observer := range observers {
		observer.RenderCallerGaveUp(ctx, targetURL, renderDuration, cause)
	}
}

func (observers RenderObservers) RenderPageTooLarge(
	ctx context.Context,
	targetURL string,
	renderDuration time.Duration,
	cause error,
) {
	for _, observer := range observers {
		observer.RenderPageTooLarge(ctx, targetURL, renderDuration, cause)
	}
}

func (observers RenderObservers) RenderFailed(
	ctx context.Context,
	targetURL string,
	renderDuration time.Duration,
	cause error,
) {
	for _, observer := range observers {
		observer.RenderFailed(ctx, targetURL, renderDuration, cause)
	}
}
