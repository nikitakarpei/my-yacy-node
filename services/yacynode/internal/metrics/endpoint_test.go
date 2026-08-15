package metrics_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

func TestEndpointCountsRequestsByEndpointAndStatus(t *testing.T) {
	registry := prometheus.NewRegistry()
	endpoints := metrics.NewHTTPEndpointMetrics(registry)

	endpoints.Observe("/yacy/transferRWI.html", http.StatusOK, 2*time.Millisecond)
	endpoints.Observe("/yacy/transferRWI.html", http.StatusOK, 4*time.Millisecond)
	endpoints.Observe("/yacy/transferRWI.html", http.StatusBadRequest, time.Millisecond)
	endpoints.Observe("", http.StatusNotFound, time.Millisecond)

	expected := `
# HELP http_requests_total HTTP requests served, by endpoint and response status code.
# TYPE http_requests_total counter
http_requests_total{code="200",endpoint="/yacy/transferRWI.html"} 2
http_requests_total{code="400",endpoint="/yacy/transferRWI.html"} 1
http_requests_total{code="404",endpoint="unmatched"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"http_requests_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
	if got := testutil.CollectAndCount(
		registry,
		"http_request_duration_seconds",
	); got != 2 {
		t.Errorf("timed endpoints = %v, want 2", got)
	}
}
