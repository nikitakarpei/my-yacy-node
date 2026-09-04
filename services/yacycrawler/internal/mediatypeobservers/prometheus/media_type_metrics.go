package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

type MediaTypeMetrics struct {
	mediaTypesUnparsed prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *MediaTypeMetrics {
	metrics := &MediaTypeMetrics{
		mediaTypesUnparsed: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_media_types_unparsed_total",
			Help: "Fetched pages whose content type cannot be parsed.",
		}),
	}
	registry.MustRegister(metrics.mediaTypesUnparsed)
	return metrics
}

func (metrics *MediaTypeMetrics) MediaTypeUnparsed(context.Context, string, error) {
	metrics.mediaTypesUnparsed.Inc()
}
