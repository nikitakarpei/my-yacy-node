package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageReadingMetrics struct {
	mediaTypesUnparsed            prometheusclient.Counter
	linkDiscoveryRefusalsEnforced prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *PageReadingMetrics {
	metrics := &PageReadingMetrics{
		mediaTypesUnparsed: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_media_types_unparsed_total",
			Help: "Fetched pages whose content type cannot be parsed.",
		}),
		linkDiscoveryRefusalsEnforced: prometheusclient.NewCounter(
			prometheusclient.CounterOpts{
				Name: "yacycrawler_link_discovery_refusals_enforced_total",
				Help: "Pages whose refusal of link discovery the crawler enforced.",
			},
		),
	}
	registry.MustRegister(metrics.mediaTypesUnparsed, metrics.linkDiscoveryRefusalsEnforced)
	return metrics
}

func (metrics *PageReadingMetrics) MediaTypeUnparsed(context.Context, string, error) {
	metrics.mediaTypesUnparsed.Inc()
}

func (metrics *PageReadingMetrics) LinkDiscoveryRefusalEnforced(
	context.Context, canonicalurl.CanonicalURL,
) {
	metrics.linkDiscoveryRefusalsEnforced.Inc()
}
