package contract_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendergate"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendermetrics"
)

type blockingRenderer struct {
	entered chan struct{}
	release chan struct{}
}

func (r *blockingRenderer) Render(
	ctx context.Context,
	targetURL string,
) (renderedpage.Page, error) {
	close(r.entered)
	<-r.release
	return renderedpage.Page{}, nil
}

func gateWithSlotHeld(
	t *testing.T,
) (*rendergate.Renderer, *rendermetrics.RenderMetrics, chan<- struct{}, *sync.WaitGroup) {
	t.Helper()

	metrics := rendermetrics.New()
	inner := &blockingRenderer{entered: make(chan struct{}), release: make(chan struct{})}
	gate := rendergate.New(inner, 1, time.Second, 1024, metrics)

	var wg sync.WaitGroup
	wg.Go(func() {
		_, _ = gate.Render(context.Background(), "http://example.com")
	})
	<-inner.entered

	return gate, metrics, inner.release, &wg
}

func scrapedMetrics(t *testing.T, metrics *rendermetrics.RenderMetrics) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	metrics.Handler().ServeHTTP(recorder, request)

	body, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatalf("read metrics response: %v", err)
	}
	return string(body)
}

func TestRenderWaitsForSlotWhenConcurrencyCapReached(t *testing.T) {
	gate, metrics, release, wg := gateWithSlotHeld(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = gate.Render(ctx, "http://example.com")

	close(release)
	wg.Wait()

	body := scrapedMetrics(t, metrics)
	if !strings.Contains(body, "renderproxy_render_waits_total 1") {
		t.Fatalf("metrics body does not contain wait count line:\n%s", body)
	}
}

func TestRenderReportsSlotWaitTimeout(t *testing.T) {
	gate, metrics, release, wg := gateWithSlotHeld(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := gate.Render(ctx, "http://example.com"); err == nil {
		t.Fatal("expected error from render with an already cancelled context")
	}

	close(release)
	wg.Wait()

	body := scrapedMetrics(t, metrics)
	want := `renderproxy_renders_failed_total{reason="` + rendergate.ReasonSlotWaitTimeout + `"} 1`
	if !strings.Contains(body, want) {
		t.Fatalf("metrics body does not contain failure count line:\n%s", body)
	}
}
