package prometheus_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	proxyresponsedeliveryobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/proxyresponsedeliveryobservers/prometheus"
)

func TestProxyResponseDeliveryMetricsRecordDeliveryFacts(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := proxyresponsedeliveryobserversprometheus.New(registry)
	metrics.ProxyResponseDelivered(context.Background(), "https://example.com")
	metrics.ProxyResponseDeliveryFailed(
		context.Background(),
		"https://example.com",
		errors.New("failed"),
	)
	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder, httptest.NewRequestWithContext(context.Background(), "GET", "/metrics", nil),
	)
	for _, expected := range []string{
		"renderproxy_proxy_responses_processed_total{outcome=\"delivered\"} 1",
		"renderproxy_proxy_responses_processed_total{outcome=\"delivery_failed\"} 1",
	} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Fatalf("metrics output does not contain %q:\n%s", expected, recorder.Body.String())
		}
	}
}
