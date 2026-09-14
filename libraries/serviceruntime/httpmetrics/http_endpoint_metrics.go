package httpmetrics

import (
	"context"
	"strconv"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
)

const (
	labelEndpoint     = "endpoint"
	labelStatusCode   = "code"
	unmatchedEndpoint = "unmatched"
)

type EndpointMetrics struct {
	requests  *prometheusclient.CounterVec
	durations *prometheusclient.HistogramVec
}

func NewEndpointMetrics(
	registry prometheusclient.Registerer,
	serviceMetricNamespace string,
) *EndpointMetrics {
	metrics := &EndpointMetrics{
		requests: prometheusclient.NewCounterVec(
			prometheusclient.CounterOpts{
				Name: serviceMetricNamespace + "_http_requests_total",
				Help: "HTTP requests served, by endpoint and response status code.",
			},
			[]string{labelEndpoint, labelStatusCode},
		),
		durations: prometheusclient.NewHistogramVec(
			prometheusclient.HistogramOpts{
				Name:    serviceMetricNamespace + "_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds, by endpoint.",
				Buckets: prometheusclient.DefBuckets,
			},
			[]string{labelEndpoint},
		),
	}
	registry.MustRegister(metrics.requests, metrics.durations)
	return metrics
}

func (m *EndpointMetrics) ObserveRequest(
	_ context.Context,
	served httpobservation.ServedRequest,
) {
	endpoint := served.Pattern
	if endpoint == "" {
		endpoint = unmatchedEndpoint
	}
	m.requests.WithLabelValues(endpoint, strconv.Itoa(served.Status)).Inc()
	m.durations.WithLabelValues(endpoint).Observe(served.Duration.Seconds())
}
