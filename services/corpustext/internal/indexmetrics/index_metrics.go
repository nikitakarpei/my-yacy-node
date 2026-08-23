package indexmetrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type IndexMetrics struct {
	scrapeRequestsReceived prometheus.Counter
	pagesIndexed           prometheus.Counter
	scrapeFailures         prometheus.Counter
	indexFailures          prometheus.Counter
	indexDurationSecs      prometheus.Histogram
}

func New(registry prometheus.Registerer) *IndexMetrics {
	metrics := &IndexMetrics{
		scrapeRequestsReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpustext_scrape_requests_received_total",
			Help: "Scrape requests received for scraping.",
		}),
		pagesIndexed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpustext_pages_indexed_total",
			Help: "Scraped pages written to the search index.",
		}),
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpustext_scrape_failures_total",
			Help: "Scrapes that failed and returned the scrape request for redelivery.",
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
		metrics.scrapeRequestsReceived,
		metrics.pagesIndexed,
		metrics.scrapeFailures,
		metrics.indexFailures,
		metrics.indexDurationSecs,
	)
	return metrics
}

func (m *IndexMetrics) ScrapeRequestReceived() { m.scrapeRequestsReceived.Inc() }
func (m *IndexMetrics) PageIndexed()           { m.pagesIndexed.Inc() }
func (m *IndexMetrics) ScrapeFailed()          { m.scrapeFailures.Inc() }
func (m *IndexMetrics) IndexFailed()           { m.indexFailures.Inc() }

func (m *IndexMetrics) IndexObserved(elapsed time.Duration) {
	m.indexDurationSecs.Observe(elapsed.Seconds())
}
