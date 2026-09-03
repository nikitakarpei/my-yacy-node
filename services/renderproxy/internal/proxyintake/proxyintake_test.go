package proxyintake_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/pagefreshness"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/proxyintake"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

type stubRenderer struct {
	page renderedpage.Page
	err  error
}

func newProxyEndpoint(renderer renderedpage.Renderer) *proxyintake.ProxyEndpoint {
	return proxyintake.New(renderer, proxyintake.ProxyResponseDeliveryObservers{})
}

func (s stubRenderer) Render(
	context.Context,
	renderedpage.Target,
) (renderedpage.Page, error) {
	return s.page, s.err
}

func TestServeHTTPRefusesConnect(t *testing.T) {
	handler := newProxyEndpoint(stubRenderer{})
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodConnect,
		"http://example.com/",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestServeHTTPRejectsNonAbsoluteRequest(t *testing.T) {
	handler := newProxyEndpoint(stubRenderer{})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/path", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServeHTTPReturnsRenderedPage(t *testing.T) {
	handler := newProxyEndpoint(stubRenderer{page: renderedpage.Page{
		StatusCode:  http.StatusOK,
		ContentType: "text/html",
		Body:        []byte("<html>hi</html>"),
	}})
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://example.com/",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "<html>hi</html>" {
		t.Fatalf("body = %q", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html" {
		t.Fatalf("content-type = %q", got)
	}
}

func TestServeHTTPRelaysRedirectLocation(t *testing.T) {
	handler := newProxyEndpoint(stubRenderer{page: renderedpage.Page{
		StatusCode: http.StatusMovedPermanently,
		Location:   "https://example.com/final",
	}})
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://example.com/",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if got := rec.Header().Get("Location"); got != "https://example.com/final" {
		t.Fatalf("location = %q", got)
	}
}

type failingResponseWriter struct {
	header http.Header
	code   int
}

func (w *failingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}
func (w *failingResponseWriter) WriteHeader(code int)      { w.code = code }
func (w *failingResponseWriter) Write([]byte) (int, error) { return 0, errors.New("client gone") }

type recordingProxyResponseDelivery struct {
	proxyResponsesDelivered    int
	proxyResponsesNotDelivered int
}

func (p *recordingProxyResponseDelivery) ProxyResponseDelivered(context.Context, string) {
	p.proxyResponsesDelivered++
}

func (p *recordingProxyResponseDelivery) ProxyResponseDeliveryFailed(
	context.Context,
	string,
	error,
) {
	p.proxyResponsesNotDelivered++
}

func TestServeHTTPSurvivesWriteFailure(t *testing.T) {
	delivery := &recordingProxyResponseDelivery{}
	handler := proxyintake.New(
		stubRenderer{page: renderedpage.Page{
			StatusCode: http.StatusOK,
			Body:       []byte("<html>hi</html>"),
		}},
		delivery,
	)
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://example.com/",
		nil,
	)
	w := &failingResponseWriter{}

	handler.ServeHTTP(w, req)

	if w.code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.code, http.StatusOK)
	}
	if delivery.proxyResponsesNotDelivered != 1 || delivery.proxyResponsesDelivered != 0 {
		t.Fatalf(
			"delivered = %d, not delivered = %d",
			delivery.proxyResponsesDelivered,
			delivery.proxyResponsesNotDelivered,
		)
	}
}

func TestServeHTTPReportsProxyResponseDelivered(t *testing.T) {
	delivery := &recordingProxyResponseDelivery{}
	handler := proxyintake.New(
		stubRenderer{page: renderedpage.Page{StatusCode: http.StatusOK}},
		delivery,
	)
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "http://example.com/", nil,
	)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if delivery.proxyResponsesDelivered != 1 || delivery.proxyResponsesNotDelivered != 0 {
		t.Fatalf(
			"delivered = %d, not delivered = %d",
			delivery.proxyResponsesDelivered,
			delivery.proxyResponsesNotDelivered,
		)
	}
}

func TestServeHTTPFailsOnDeadlineExceeded(t *testing.T) {
	handler := newProxyEndpoint(stubRenderer{err: context.DeadlineExceeded})
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://example.com/",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusGatewayTimeout)
	}
}

func TestServeHTTPFailsOnRenderError(t *testing.T) {
	handler := newProxyEndpoint(stubRenderer{err: errors.New("browser unreachable")})
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://example.com/",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

func TestServeHTTPStatesOriginReuseTermsOnTheClientResponse(t *testing.T) {
	originResponseHeader := http.Header{}
	originResponseHeader.Set("ETag", `"v1"`)
	handler := newProxyEndpoint(stubRenderer{page: renderedpage.Page{
		StatusCode: http.StatusNotModified,
		ReuseTerms: pagefreshness.ReuseTermsOf(originResponseHeader),
	}})
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://example.com/",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotModified)
	}
	if got := rec.Header().Get("ETag"); got != `"v1"` {
		t.Fatalf("etag = %q", got)
	}
}

type conditionRecordingRenderer struct {
	statedConditions http.Header
}

func (r *conditionRecordingRenderer) Render(
	_ context.Context,
	target renderedpage.Target,
) (renderedpage.Page, error) {
	r.statedConditions = http.Header{}
	target.Conditions.StateOn(r.statedConditions)
	return renderedpage.Page{StatusCode: http.StatusOK}, nil
}

func TestServeHTTPPassesClientConditionsToTheRenderer(t *testing.T) {
	renderer := &conditionRecordingRenderer{}
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://example.com/",
		nil,
	)
	req.Header.Set("If-Modified-Since", "Mon, 02 Jan 2006 15:04:05 GMT")

	newProxyEndpoint(renderer).ServeHTTP(httptest.NewRecorder(), req)

	if got := renderer.statedConditions.Get("If-Modified-Since"); got == "" {
		t.Fatal("if-modified-since did not reach the renderer")
	}
}
