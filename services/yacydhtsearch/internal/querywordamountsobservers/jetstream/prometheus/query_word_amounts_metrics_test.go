package prometheus_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	querywordamountsobserversjetstreamprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/querywordamountsobservers/jetstream/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func TestAFailedStoreIsPublishedApartFromAFailedLookup(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := querywordamountsobserversjetstreamprometheus.New(registry)
	berlin := yacymodel.WordHash("berlin")
	refused := errors.New("bucket refused")

	metrics.AmountLookupFailed(t.Context(), berlin, refused)
	metrics.AmountStoreFailed(t.Context(), berlin, refused)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_query_word_amounts_failures_total{action="lookup"} 1`,
		`yacydhtsearch_query_word_amounts_failures_total{action="store"} 1`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestEveryQueryWordAmountsFailureIsPublishedBeforeTheFirstFailure(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	querywordamountsobserversjetstreamprometheus.New(registry)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_query_word_amounts_failures_total{action="lookup"} 0`,
		`yacydhtsearch_query_word_amounts_failures_total{action="store"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
