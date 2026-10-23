package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type LinkResolutionMetrics struct {
	baseURLsUnresolved prometheusclient.Counter
	linksUnresolved    prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *LinkResolutionMetrics {
	metrics := &LinkResolutionMetrics{
		baseURLsUnresolved: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_base_urls_unresolved_total",
			Help: "Pages whose stated base URL cannot be resolved, read against the page URL instead.",
		}),
		linksUnresolved: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_links_unresolved_total",
			Help: "Links that cannot be resolved into a URL, left off the frontier.",
		}),
	}
	registry.MustRegister(metrics.baseURLsUnresolved, metrics.linksUnresolved)
	return metrics
}

func (metrics *LinkResolutionMetrics) BaseURLUnresolved(
	context.Context, canonicalurl.CanonicalURL, string, error,
) {
	metrics.baseURLsUnresolved.Inc()
}

func (metrics *LinkResolutionMetrics) LinksUnresolved(
	_ context.Context, _ canonicalurl.CanonicalURL, links int,
) {
	metrics.linksUnresolved.Add(float64(links))
}
