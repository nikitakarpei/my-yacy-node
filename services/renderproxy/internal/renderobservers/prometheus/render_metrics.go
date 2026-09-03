package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendergate"
)

const (
	labelOutcome          = "outcome"
	resultRenderSucceeded = "succeeded"
)

type RenderMetrics struct {
	rendersProcessed   *prometheusclient.CounterVec
	renderDurationSecs prometheusclient.Histogram
}

func New(registry prometheusclient.Registerer) *RenderMetrics {
	metrics := &RenderMetrics{
		rendersProcessed: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "renderproxy_renders_processed_total",
			Help: "Renders processed, by outcome.",
		}, []string{labelOutcome}),
		renderDurationSecs: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "renderproxy_render_duration_seconds",
			Help:    "Render duration in seconds.",
			Buckets: prometheusclient.DefBuckets,
		}),
	}
	registry.MustRegister(metrics.rendersProcessed, metrics.renderDurationSecs)
	return metrics
}

func (metrics *RenderMetrics) RenderSucceeded(
	_ context.Context,
	_ string,
	renderDuration time.Duration,
) {
	metrics.recordRender(resultRenderSucceeded, renderDuration)
}

func (metrics *RenderMetrics) RenderFailed(
	_ context.Context,
	_ string,
	renderDuration time.Duration,
	reason rendergate.RenderFailureReason,
	_ error,
) {
	metrics.recordRender(string(reason), renderDuration)
}

func (metrics *RenderMetrics) recordRender(outcome string, renderDuration time.Duration) {
	metrics.rendersProcessed.WithLabelValues(outcome).Inc()
	metrics.renderDurationSecs.Observe(renderDuration.Seconds())
}
