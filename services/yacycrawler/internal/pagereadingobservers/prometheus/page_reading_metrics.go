package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const labelDirective = "directive"

type PageReadingMetrics struct {
	directivesEnforced  *prometheusclient.CounterVec
	pagesHTMLUnreadable prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *PageReadingMetrics {
	metrics := &PageReadingMetrics{
		directivesEnforced: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_directives_enforced_total",
			Help: "Page directives enforced, by directive.",
		}, []string{labelDirective}),
		pagesHTMLUnreadable: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_html_unreadable_total",
			Help: "Fetched pages whose HTML cannot be read.",
		}),
	}
	registry.MustRegister(metrics.directivesEnforced, metrics.pagesHTMLUnreadable)
	return metrics
}

func (metrics *PageReadingMetrics) IndexingRefusalEnforced(
	context.Context,
	canonicalurl.CanonicalURL,
) {
	metrics.directivesEnforced.WithLabelValues("indexing").Inc()
}

func (metrics *PageReadingMetrics) LinkDiscoveryRefusalEnforced(
	context.Context, canonicalurl.CanonicalURL,
) {
	metrics.directivesEnforced.WithLabelValues("link_discovery").Inc()
}

func (metrics *PageReadingMetrics) PageHTMLUnreadable(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.pagesHTMLUnreadable.Inc()
}
