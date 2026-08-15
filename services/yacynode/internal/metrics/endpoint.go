package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelEndpoint     = "endpoint"
	labelStatusCode   = "code"
	unmatchedEndpoint = "unmatched"
)

type HTTPEndpointMetrics struct {
	requests  *prometheus.CounterVec
	durations *prometheus.HistogramVec
}

func NewHTTPEndpointMetrics(registry prometheus.Registerer) *HTTPEndpointMetrics {
	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests served, by endpoint and response status code.",
		},
		[]string{labelEndpoint, labelStatusCode},
	)
	durations := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds, by endpoint.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{labelEndpoint},
	)
	registry.MustRegister(requests, durations)

	return &HTTPEndpointMetrics{requests: requests, durations: durations}
}

func (e *HTTPEndpointMetrics) Observe(endpoint string, status int, elapsed time.Duration) {
	if endpoint == "" {
		endpoint = unmatchedEndpoint
	}
	e.requests.WithLabelValues(endpoint, strconv.Itoa(status)).Inc()
	e.durations.WithLabelValues(endpoint).Observe(elapsed.Seconds())
}
