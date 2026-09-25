package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	urlmetadataaskceilingsobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/urlmetadataaskceilingsobservers/prometheus"
)

const peerAddress = "http://10.0.0.1:8090"

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func TestEachHandedOutCeilingIsCountedInItsBucket(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := urlmetadataaskceilingsobserversprometheus.New(registry)

	metrics.AskCeilingHandedOut(t.Context(), peerAddress, 400)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_url_metadata_ask_ceiling_documents_bucket{le="200"} 0`,
		`yacydhtsearch_url_metadata_ask_ceiling_documents_bucket{le="400"} 1`,
		`yacydhtsearch_url_metadata_ask_ceiling_documents_count 1`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
