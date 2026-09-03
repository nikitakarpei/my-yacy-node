package rendergate_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendergate"
)

type stubRenderer struct {
	err   error
	delay time.Duration
}

func (s *stubRenderer) Render(
	ctx context.Context,
	_ renderedpage.Target,
) (renderedpage.Page, error) {
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return renderedpage.Page{}, fmt.Errorf("stub render canceled: %w", ctx.Err())
	}
	return renderedpage.Page{}, s.err
}

type recordingRenderObserver struct {
	succeeded     int
	failed        int
	failureReason rendergate.RenderFailureReason
}

func (o *recordingRenderObserver) RenderSucceeded(context.Context, string, time.Duration) {
	o.succeeded++
}

func (o *recordingRenderObserver) RenderFailed(
	_ context.Context,
	_ string,
	_ time.Duration,
	reason rendergate.RenderFailureReason,
	_ error,
) {
	o.failed++
	o.failureReason = reason
}

func TestDeadlineRendererReportsOversizedPageFailureReason(t *testing.T) {
	observer := &recordingRenderObserver{}
	renderer := rendergate.NewDeadlineRenderer(
		&stubRenderer{err: fmt.Errorf("serialize: %w", renderedpage.ErrTooLarge)},
		time.Second,
		observer,
	)

	_, err := renderer.Render(t.Context(), renderedpage.Target{URL: "https://example.com"})
	if err == nil {
		t.Fatal("expected oversized page error")
	}
	if observer.failureReason != rendergate.RenderFailurePageTooLarge {
		t.Fatalf("failure reason = %q", observer.failureReason)
	}
}

func TestDeadlineRendererReportsDeadlineFailureReason(t *testing.T) {
	observer := &recordingRenderObserver{}
	renderer := rendergate.NewDeadlineRenderer(
		&stubRenderer{delay: 50 * time.Millisecond},
		5*time.Millisecond,
		observer,
	)

	_, err := renderer.Render(t.Context(), renderedpage.Target{URL: "https://example.com"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline exceeded", err)
	}
	if observer.failureReason != rendergate.RenderFailureTimedOut {
		t.Fatalf("failure reason = %q", observer.failureReason)
	}
}

func TestDeadlineRendererReportsSuccess(t *testing.T) {
	observer := &recordingRenderObserver{}
	renderer := rendergate.NewDeadlineRenderer(&stubRenderer{}, time.Second, observer)

	_, err := renderer.Render(t.Context(), renderedpage.Target{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if observer.succeeded != 1 || observer.failed != 0 {
		t.Fatalf("succeeded = %d, failed = %d", observer.succeeded, observer.failed)
	}
}

type blockingRenderer struct {
	entered chan struct{}
	release chan struct{}
}

func (r *blockingRenderer) Render(
	context.Context,
	renderedpage.Target,
) (renderedpage.Page, error) {
	r.entered <- struct{}{}
	<-r.release
	return renderedpage.Page{}, nil
}

type recordingRenderCapacityObserver struct {
	waitedForCapacity       int
	endedWaitingForCapacity int
}

func (o *recordingRenderCapacityObserver) RenderWaitedForCapacity(
	context.Context,
	string,
	time.Duration,
) {
	o.waitedForCapacity++
}

func (o *recordingRenderCapacityObserver) RenderEndedWhileWaitingForCapacity(
	context.Context,
	string,
	time.Duration,
	error,
) {
	o.endedWaitingForCapacity++
}

func TestCapacityLimitedRendererReportsRenderEndedWhileWaitingForCapacity(t *testing.T) {
	inner := &blockingRenderer{entered: make(chan struct{}, 1), release: make(chan struct{})}
	observer := &recordingRenderCapacityObserver{}
	renderer := rendergate.NewCapacityLimitedRenderer(inner, 1, observer)

	var held sync.WaitGroup
	held.Go(func() {
		_, _ = renderer.Render(t.Context(), renderedpage.Target{URL: "https://held.example"})
	})
	<-inner.entered

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := renderer.Render(ctx, renderedpage.Target{URL: "https://waiting.example"})
	close(inner.release)
	held.Wait()

	if err == nil {
		t.Fatal("expected render capacity error")
	}
	if observer.endedWaitingForCapacity != 1 || observer.waitedForCapacity != 0 {
		t.Fatalf(
			"ended while waiting = %d, waited for capacity = %d",
			observer.endedWaitingForCapacity,
			observer.waitedForCapacity,
		)
	}
}
