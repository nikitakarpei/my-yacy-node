package pagecacheproxy

import "github.com/prometheus/client_golang/prometheus"

type pageFetchMetrics struct {
	pageFetches *prometheus.CounterVec
}

func pageFetchMetricsFor(registry prometheus.Registerer) *pageFetchMetrics {
	metrics := &pageFetchMetrics{
		pageFetches: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "yacypeer_page_fetches_total",
			Help: "Page fetches that the page cache proxy answered, by source: cache or origin.",
		}, []string{"source"}),
	}
	for _, source := range []string{sourceCache, sourceOrigin} {
		metrics.pageFetches.WithLabelValues(source)
	}
	registry.MustRegister(metrics.pageFetches)
	return metrics
}

func (metrics *pageFetchMetrics) record(source string) {
	metrics.pageFetches.WithLabelValues(source).Inc()
}
