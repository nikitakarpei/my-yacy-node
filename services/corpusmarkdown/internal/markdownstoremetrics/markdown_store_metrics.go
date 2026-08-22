package markdownstoremetrics

import "github.com/prometheus/client_golang/prometheus"

type MarkdownStoreMetrics struct {
	pagesReceived  prometheus.Counter
	pagesStored    prometheus.Counter
	scrapeFailures prometheus.Counter
	storeFailures  prometheus.Counter
}

func New(registry prometheus.Registerer) *MarkdownStoreMetrics {
	metrics := &MarkdownStoreMetrics{
		pagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_pages_received_total",
			Help: "Scrape requests received for scraping.",
		}),
		pagesStored: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_pages_stored_total",
			Help: "Page markdown written to the object store.",
		}),
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_scrape_failures_total",
			Help: "Scrapes that failed and returned the scrape request for redelivery.",
		}),
		storeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_store_failures_total",
			Help: "Object-store writes that failed and returned the page for redelivery.",
		}),
	}
	registry.MustRegister(
		metrics.pagesReceived,
		metrics.pagesStored,
		metrics.scrapeFailures,
		metrics.storeFailures,
	)
	return metrics
}

func (m *MarkdownStoreMetrics) PageReceived() { m.pagesReceived.Inc() }
func (m *MarkdownStoreMetrics) PageStored()   { m.pagesStored.Inc() }
func (m *MarkdownStoreMetrics) ScrapeFailed() { m.scrapeFailures.Inc() }
func (m *MarkdownStoreMetrics) StoreFailed()  { m.storeFailures.Inc() }
