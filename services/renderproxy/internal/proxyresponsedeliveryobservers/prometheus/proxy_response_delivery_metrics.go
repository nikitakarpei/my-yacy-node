package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

const (
	labelOutcome                = "outcome"
	proxyResponseDelivered      = "delivered"
	proxyResponseDeliveryFailed = "delivery_failed"
)

type ProxyResponseDeliveryMetrics struct {
	proxyResponsesProcessed *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *ProxyResponseDeliveryMetrics {
	metrics := &ProxyResponseDeliveryMetrics{
		proxyResponsesProcessed: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "renderproxy_proxy_responses_processed_total",
			Help: "Proxy responses processed, by outcome.",
		}, []string{labelOutcome}),
	}
	registry.MustRegister(metrics.proxyResponsesProcessed)
	return metrics
}

func (metrics *ProxyResponseDeliveryMetrics) ProxyResponseDelivered(
	context.Context,
	string,
) {
	metrics.proxyResponsesProcessed.WithLabelValues(proxyResponseDelivered).Inc()
}

func (metrics *ProxyResponseDeliveryMetrics) ProxyResponseDeliveryFailed(
	context.Context,
	string,
	error,
) {
	metrics.proxyResponsesProcessed.WithLabelValues(proxyResponseDeliveryFailed).Inc()
}
