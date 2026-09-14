package httpmetrics_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
)

func TestEndpointMetricsCountRequestsByEndpointAndStatus(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := httpmetrics.NewEndpointMetrics(registry, "service")

	metrics.ObserveRequest(context.Background(), httpobservation.ServedRequest{
		Pattern: "/endpoint", Status: http.StatusOK, Duration: 2 * time.Millisecond,
	})
	metrics.ObserveRequest(context.Background(), httpobservation.ServedRequest{
		Pattern: "/endpoint", Status: http.StatusOK, Duration: 4 * time.Millisecond,
	})
	metrics.ObserveRequest(context.Background(), httpobservation.ServedRequest{
		Pattern: "/endpoint", Status: http.StatusBadRequest, Duration: time.Millisecond,
	})
	metrics.ObserveRequest(context.Background(), httpobservation.ServedRequest{
		Status: http.StatusNotFound, Duration: time.Millisecond,
	})

	expected := `
# HELP service_http_requests_total HTTP requests served, by endpoint and response status code.
# TYPE service_http_requests_total counter
service_http_requests_total{code="200",endpoint="/endpoint"} 2
service_http_requests_total{code="400",endpoint="/endpoint"} 1
service_http_requests_total{code="404",endpoint="unmatched"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"service_http_requests_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
	if got := testutil.CollectAndCount(
		registry,
		"service_http_request_duration_seconds",
	); got != 2 {
		t.Errorf("timed endpoints = %v, want 2", got)
	}
}
