package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type LinkResolutionMetrics struct {
	baseHrefsUnresolved prometheusclient.Counter
	linkHrefsUnresolved prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *LinkResolutionMetrics {
	metrics := &LinkResolutionMetrics{
		baseHrefsUnresolved: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_base_hrefs_unresolved_total",
			Help: "Pages whose base href cannot be resolved, read against the page url.",
		}),
		linkHrefsUnresolved: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_link_hrefs_unresolved_total",
			Help: "Link hrefs that cannot be resolved, left off the frontier.",
		}),
	}
	registry.MustRegister(metrics.baseHrefsUnresolved, metrics.linkHrefsUnresolved)
	return metrics
}

func (metrics *LinkResolutionMetrics) BaseHrefUnresolved(
	context.Context, canonicalurl.CanonicalURL, string, error,
) {
	metrics.baseHrefsUnresolved.Inc()
}

func (metrics *LinkResolutionMetrics) LinkHrefsUnresolved(
	_ context.Context, _ canonicalurl.CanonicalURL, hrefs int,
) {
	metrics.linkHrefsUnresolved.Add(float64(hrefs))
}
