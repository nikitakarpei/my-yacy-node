package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

const (
	labelOutcome              = "outcome"
	resultRenderSucceeded     = "succeeded"
	resultRenderTimedOut      = "timed_out"
	resultRenderCallerGaveUp  = "caller_gave_up"
	resultRenderPageTooLarge  = "page_too_large"
	resultRenderFailedUnknown = "unexpected"
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

func (metrics *RenderMetrics) RenderTimedOut(
	_ context.Context,
	_ string,
	renderDuration time.Duration,
	_ error,
) {
	metrics.recordRender(resultRenderTimedOut, renderDuration)
}

func (metrics *RenderMetrics) RenderCallerGaveUp(
	_ context.Context,
	_ string,
	renderDuration time.Duration,
	_ error,
) {
	metrics.recordRender(resultRenderCallerGaveUp, renderDuration)
}

func (metrics *RenderMetrics) RenderPageTooLarge(
	_ context.Context,
	_ string,
	renderDuration time.Duration,
	_ error,
) {
	metrics.recordRender(resultRenderPageTooLarge, renderDuration)
}

func (metrics *RenderMetrics) RenderFailed(
	_ context.Context,
	_ string,
	renderDuration time.Duration,
	_ error,
) {
	metrics.recordRender(resultRenderFailedUnknown, renderDuration)
}

func (metrics *RenderMetrics) recordRender(outcome string, renderDuration time.Duration) {
	metrics.rendersProcessed.WithLabelValues(outcome).Inc()
	metrics.renderDurationSecs.Observe(renderDuration.Seconds())
}
