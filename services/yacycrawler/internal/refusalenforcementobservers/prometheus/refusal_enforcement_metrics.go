package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type RefusalEnforcementMetrics struct {
	linkDiscoveryRefusalsEnforced prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *RefusalEnforcementMetrics {
	metrics := &RefusalEnforcementMetrics{
		linkDiscoveryRefusalsEnforced: prometheusclient.NewCounter(
			prometheusclient.CounterOpts{
				Name: "yacycrawler_link_discovery_refusals_enforced_total",
				Help: "Pages whose refusal of link discovery the crawler enforced.",
			},
		),
	}
	registry.MustRegister(metrics.linkDiscoveryRefusalsEnforced)
	return metrics
}

func (metrics *RefusalEnforcementMetrics) LinkDiscoveryRefusalEnforced(
	context.Context, canonicalurl.CanonicalURL,
) {
	metrics.linkDiscoveryRefusalsEnforced.Inc()
}
