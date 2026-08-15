package rendermetrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const labelReason = "reason"

type RenderMetrics struct {
	rendersSucceeded   prometheus.Counter
	rendersFailed      *prometheus.CounterVec
	renderWaits        prometheus.Counter
	renderDurationSecs prometheus.Histogram
}

func New(registry prometheus.Registerer) *RenderMetrics {
	metrics := &RenderMetrics{
		rendersSucceeded: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "renderproxy_renders_succeeded_total",
			Help: "Pages returned after a settled render.",
		}),
		rendersFailed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "renderproxy_renders_failed_total",
			Help: "Renders that failed, by reason.",
		}, []string{labelReason}),
		renderWaits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "renderproxy_render_waits_total",
			Help: "Requests that waited for a render slot under the concurrency cap.",
		}),
		renderDurationSecs: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "renderproxy_render_duration_seconds",
			Help:    "Render duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
	}
	registry.MustRegister(
		metrics.rendersSucceeded,
		metrics.rendersFailed,
		metrics.renderWaits,
		metrics.renderDurationSecs,
	)
	return metrics
}

func (m *RenderMetrics) RenderSucceeded() { m.rendersSucceeded.Inc() }
func (m *RenderMetrics) RenderFailed(reason string) {
	m.rendersFailed.WithLabelValues(reason).Inc()
}
func (m *RenderMetrics) RenderWaited() { m.renderWaits.Inc() }

func (m *RenderMetrics) RenderObserved(elapsed time.Duration) {
	m.renderDurationSecs.Observe(elapsed.Seconds())
}
