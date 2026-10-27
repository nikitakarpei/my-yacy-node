package webuiproxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuiproxy"
)

func webUIURLOf(t *testing.T, webUI *httptest.Server) *url.URL {
	t.Helper()
	webUIURL, err := url.Parse(webUI.URL)
	if err != nil {
		t.Fatalf("parse the web UI URL: %v", err)
	}
	return webUIURL
}

func TestToForwardsThePathTheQueryAndTheHostTheBrowserAskedFor(t *testing.T) {
	webUI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString( //nolint:gosec // G705: the test echoes what the web UI received.
			w,
			r.Host+" "+r.URL.RequestURI()+" "+r.Header.Get("X-Forwarded-Proto"),
		)
	}))
	t.Cleanup(webUI.Close)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"http://peer.example/crawlconsole/crawls/c1?hosts=h1", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	answer := httptest.NewRecorder()

	webuiproxy.To(webUIURLOf(t, webUI)).ServeHTTP(answer, request)

	if body := answer.Body.String(); body != "peer.example /crawlconsole/crawls/c1?hosts=h1 https" {
		t.Errorf("web UI saw %q, want the host, path, query and scheme of the browser", body)
	}
}

func TestToAnswersBadGatewayWhenTheWebUIDoesNotAnswer(t *testing.T) {
	webUI := httptest.NewServer(http.NotFoundHandler())
	webUIURL := webUIURLOf(t, webUI)
	webUI.Close()
	answer := httptest.NewRecorder()

	webuiproxy.To(webUIURL).ServeHTTP(answer,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://peer.example/", nil))

	if answer.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", answer.Code, http.StatusBadGateway)
	}
}
