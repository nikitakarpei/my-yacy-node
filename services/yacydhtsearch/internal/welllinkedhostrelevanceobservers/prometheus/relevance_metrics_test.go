package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	welllinkedhostrelevanceobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/welllinkedhostrelevanceobservers/prometheus"
)

func TestTheDemotedDocumentsOfEveryQueryAddUp(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := welllinkedhostrelevanceobserversprometheus.New(registry)

	metrics.DocumentsDemoted(t.Context(), 3)
	metrics.DocumentsDemoted(t.Context(), 0)
	metrics.DocumentsDemoted(t.Context(), 4)

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)
	if body := recorder.Body.String(); !strings.Contains(
		body, "yacydhtsearch_well_linked_host_demoted_documents_total 7",
	) {
		t.Fatalf("metrics do not carry the 7 demoted documents:\n%s", body)
	}
}
