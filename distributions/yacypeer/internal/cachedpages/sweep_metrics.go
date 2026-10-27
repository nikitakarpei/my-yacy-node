package cachedpages

import "github.com/prometheus/client_golang/prometheus"

type sweepMetrics struct {
	cachedPageBytes prometheus.Gauge
	evictedPages    prometheus.Counter
}

func sweepMetricsFor(registry prometheus.Registerer) *sweepMetrics {
	metrics := &sweepMetrics{
		cachedPageBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "yacypeer_cached_page_bytes",
			Help: "Bytes that the cached pages take on disk, at the last sweep.",
		}),
		evictedPages: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yacypeer_cached_pages_evicted_total",
			Help: "Cached pages removed before their retention ended, to stay within the byte maximum.",
		}),
	}
	registry.MustRegister(metrics.cachedPageBytes, metrics.evictedPages)
	return metrics
}

func (metrics *sweepMetrics) record(sumOfCachedBytes int64, amountOfEvictedPages int) {
	metrics.cachedPageBytes.Set(float64(sumOfCachedBytes))
	metrics.evictedPages.Add(float64(amountOfEvictedPages))
}
