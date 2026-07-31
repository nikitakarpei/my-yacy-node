package indexmetrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type IndexMetrics struct {
	registry          *prometheus.Registry
	pagesReceived     prometheus.Counter
	pagesIndexed      prometheus.Counter
	indexFailures     prometheus.Counter
	indexDurationSecs prometheus.Histogram
}

func New() *IndexMetrics {
	registry := prometheus.NewRegistry()
	metrics := &IndexMetrics{
		registry: registry,
		pagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpustext_pages_received_total",
			Help: "Crawled pages received for indexing.",
		}),
		pagesIndexed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpustext_pages_indexed_total",
			Help: "Crawled pages written to the search index.",
		}),
		indexFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpustext_index_failures_total",
			Help: "Index writes that failed and returned the page for redelivery.",
		}),
		indexDurationSecs: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "corpustext_index_duration_seconds",
			Help:    "Search-index write duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
	}
	registry.MustRegister(
		metrics.pagesReceived,
		metrics.pagesIndexed,
		metrics.indexFailures,
		metrics.indexDurationSecs,
	)
	return metrics
}

func (m *IndexMetrics) PageReceived() { m.pagesReceived.Inc() }
func (m *IndexMetrics) PageIndexed()  { m.pagesIndexed.Inc() }
func (m *IndexMetrics) IndexFailed()  { m.indexFailures.Inc() }

func (m *IndexMetrics) IndexObserved(elapsed time.Duration) {
	m.indexDurationSecs.Observe(elapsed.Seconds())
}

func (m *IndexMetrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
