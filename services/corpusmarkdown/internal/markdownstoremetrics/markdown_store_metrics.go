package markdownstoremetrics

import "github.com/prometheus/client_golang/prometheus"

type MarkdownStoreMetrics struct {
	pagesReceived prometheus.Counter
	pagesStored   prometheus.Counter
	storeFailures prometheus.Counter
}

func New(registry prometheus.Registerer) *MarkdownStoreMetrics {
	metrics := &MarkdownStoreMetrics{
		pagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_pages_received_total",
			Help: "Crawled page markdown representations received for storage.",
		}),
		pagesStored: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_pages_stored_total",
			Help: "Markdown representations written to the object store.",
		}),
		storeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_store_failures_total",
			Help: "Object-store writes that failed and returned the page for redelivery.",
		}),
	}
	registry.MustRegister(
		metrics.pagesReceived,
		metrics.pagesStored,
		metrics.storeFailures,
	)
	return metrics
}

func (m *MarkdownStoreMetrics) PageReceived() { m.pagesReceived.Inc() }
func (m *MarkdownStoreMetrics) PageStored()   { m.pagesStored.Inc() }
func (m *MarkdownStoreMetrics) StoreFailed()  { m.storeFailures.Inc() }
