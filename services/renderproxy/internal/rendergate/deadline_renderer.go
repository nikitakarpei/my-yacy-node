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
	RenderFailed(
		ctx context.Context,
		targetURL string,
		renderDuration time.Duration,
		reason RenderFailureReason,
		cause error,
	)
}

type RenderFailureReason string

const (
	RenderFailureUnexpected   RenderFailureReason = "unexpected"
	RenderFailureTimedOut     RenderFailureReason = "timed_out"
	RenderFailureCallerGaveUp RenderFailureReason = "caller_gave_up"
	RenderFailurePageTooLarge RenderFailureReason = "page_too_large"
)

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
		r.renderObserver.RenderFailed(
			ctx, target.URL, renderDuration, renderFailureReasonFrom(ctx, err), err,
		)
		return renderedpage.Page{}, fmt.Errorf("render %s: %w", target.URL, err)
	}
	r.renderObserver.RenderSucceeded(ctx, target.URL, renderDuration)
	return page, nil
}

func renderFailureReasonFrom(callerContext context.Context, cause error) RenderFailureReason {
	if callerContext.Err() != nil {
		return RenderFailureCallerGaveUp
	}
	if errors.Is(cause, context.DeadlineExceeded) {
		return RenderFailureTimedOut
	}
	if errors.Is(cause, renderedpage.ErrTooLarge) {
		return RenderFailurePageTooLarge
	}
	return RenderFailureUnexpected
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

func (observers RenderObservers) RenderFailed(
	ctx context.Context,
	targetURL string,
	renderDuration time.Duration,
	reason RenderFailureReason,
	cause error,
) {
	for _, observer := range observers {
		observer.RenderFailed(ctx, targetURL, renderDuration, reason, cause)
	}
}
