package rendergate

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

type RenderCapacityObserver interface {
	RenderWaitedForCapacity(ctx context.Context, targetURL string, waitDuration time.Duration)
	RenderEndedWhileWaitingForCapacity(
		ctx context.Context,
		targetURL string,
		waitDuration time.Duration,
		cause error,
	)
}

type CapacityLimitedRenderer struct {
	inner                  renderedpage.Renderer
	renderCapacity         chan struct{}
	renderCapacityObserver RenderCapacityObserver
}

func NewCapacityLimitedRenderer(
	inner renderedpage.Renderer,
	renderConcurrency int,
	renderCapacityObserver RenderCapacityObserver,
) *CapacityLimitedRenderer {
	return &CapacityLimitedRenderer{
		inner:                  inner,
		renderCapacity:         make(chan struct{}, renderConcurrency),
		renderCapacityObserver: renderCapacityObserver,
	}
}

func (r *CapacityLimitedRenderer) Render(
	ctx context.Context,
	target renderedpage.Target,
) (renderedpage.Page, error) {
	waitDuration, waited, err := r.acquireRenderCapacity(ctx)
	if err != nil {
		r.renderCapacityObserver.RenderEndedWhileWaitingForCapacity(
			ctx, target.URL, waitDuration, err,
		)
		return renderedpage.Page{}, fmt.Errorf("acquire render capacity: %w", err)
	}
	if waited {
		r.renderCapacityObserver.RenderWaitedForCapacity(
			ctx, target.URL, waitDuration,
		)
	}
	defer func() { <-r.renderCapacity }()

	return r.inner.Render(ctx, target)
}

type RenderCapacityObservers []RenderCapacityObserver

func (observers RenderCapacityObservers) RenderWaitedForCapacity(
	ctx context.Context,
	targetURL string,
	waitDuration time.Duration,
) {
	for _, observer := range observers {
		observer.RenderWaitedForCapacity(ctx, targetURL, waitDuration)
	}
}

func (observers RenderCapacityObservers) RenderEndedWhileWaitingForCapacity(
	ctx context.Context,
	targetURL string,
	waitDuration time.Duration,
	cause error,
) {
	for _, observer := range observers {
		observer.RenderEndedWhileWaitingForCapacity(ctx, targetURL, waitDuration, cause)
	}
}

func (r *CapacityLimitedRenderer) acquireRenderCapacity(
	ctx context.Context,
) (time.Duration, bool, error) {
	select {
	case r.renderCapacity <- struct{}{}:
		return 0, false, nil
	default:
	}

	waitStarted := time.Now()
	select {
	case r.renderCapacity <- struct{}{}:
		return time.Since(waitStarted), true, nil
	case <-ctx.Done():
		return time.Since(waitStarted), true, ctx.Err()
	}
}
