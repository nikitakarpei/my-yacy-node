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
	succeeded    int
	timedOut     int
	callerGaveUp int
	pageTooLarge int
	failed       int
}

func (o *recordingRenderObserver) RenderSucceeded(context.Context, string, time.Duration) {
	o.succeeded++
}

func (o *recordingRenderObserver) RenderTimedOut(context.Context, string, time.Duration, error) {
	o.timedOut++
}

func (o *recordingRenderObserver) RenderCallerGaveUp(
	context.Context,
	string,
	time.Duration,
	error,
) {
	o.callerGaveUp++
}

func (o *recordingRenderObserver) RenderPageTooLarge(
	context.Context,
	string,
	time.Duration,
	error,
) {
	o.pageTooLarge++
}

func (o *recordingRenderObserver) RenderFailed(context.Context, string, time.Duration, error) {
	o.failed++
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
	if observer.pageTooLarge != 1 {
		t.Fatalf("page too large = %d", observer.pageTooLarge)
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
	if observer.timedOut != 1 {
		t.Fatalf("timed out = %d", observer.timedOut)
	}
}

func TestDeadlineRendererTellsACallerThatGaveUpFromItsOwnDeadline(t *testing.T) {
	observer := &recordingRenderObserver{}
	renderer := rendergate.NewDeadlineRenderer(
		&stubRenderer{delay: time.Minute},
		time.Minute,
		observer,
	)
	ctx, giveUp := context.WithCancel(context.Background())
	giveUp()

	if _, err := renderer.Render(
		ctx, renderedpage.Target{URL: "https://example.com"},
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the cancellation", err)
	}
	if observer.callerGaveUp != 1 {
		t.Fatalf("caller gave up = %d, want 1", observer.callerGaveUp)
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
