package prometheus_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	peerlivenessobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenessobservers/prometheus"
)

const probedAddress = "http://10.0.0.1:8090"

var errProbe = errors.New("the address did not answer")

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func TestEveryReasonAProbeFoundNoPeerIsPublishedApart(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peerlivenessobserversprometheus.New(registry)

	metrics.ProbeCouldNotBeBuilt(t.Context(), probedAddress, errProbe)
	metrics.PeerDidNotAnswerTheProbe(t.Context(), probedAddress, errProbe)
	metrics.PeerRefusedTheProbe(t.Context(), probedAddress, http.StatusForbidden)
	metrics.ProbeAnswerCouldNotBeRead(t.Context(), probedAddress, errProbe)
	metrics.ProbeAnswerCarriedNoRWICount(t.Context(), probedAddress, errProbe)

	body := publishedBy(t, registry)
	for _, failure := range []string{
		"unusableAddress", "noAnswer", "refused", "answerIncomplete", "noRWICount",
	} {
		published := `yacydhtsearch_probe_failures_total{failure="` + failure + `"} 1`
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestEveryReasonAProbeFoundNoPeerIsPublishedBeforeTheFirstProbe(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	peerlivenessobserversprometheus.New(registry)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_probe_failures_total{failure="unusableAddress"} 0`,
		`yacydhtsearch_probe_failures_total{failure="noAnswer"} 0`,
		`yacydhtsearch_probe_failures_total{failure="refused"} 0`,
		`yacydhtsearch_probe_failures_total{failure="answerIncomplete"} 0`,
		`yacydhtsearch_probe_failures_total{failure="noRWICount"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
