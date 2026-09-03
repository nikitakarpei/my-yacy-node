package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

type RenderCapacityMetrics struct {
	renderCapacityWaitSecs         prometheusclient.Histogram
	rendersEndedWaitingForCapacity prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *RenderCapacityMetrics {
	metrics := &RenderCapacityMetrics{
		renderCapacityWaitSecs: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "renderproxy_render_capacity_wait_seconds",
			Help:    "Time renders spent waiting for capacity in seconds.",
			Buckets: prometheusclient.DefBuckets,
		}),
		rendersEndedWaitingForCapacity: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "renderproxy_renders_ended_while_waiting_for_capacity_total",
			Help: "Renders that ended before capacity became available.",
		}),
	}
	registry.MustRegister(metrics.renderCapacityWaitSecs, metrics.rendersEndedWaitingForCapacity)
	return metrics
}

func (metrics *RenderCapacityMetrics) RenderWaitedForCapacity(
	_ context.Context,
	_ string,
	waitDuration time.Duration,
) {
	metrics.renderCapacityWaitSecs.Observe(waitDuration.Seconds())
}

func (metrics *RenderCapacityMetrics) RenderEndedWhileWaitingForCapacity(
	context.Context,
	string,
	time.Duration,
	error,
) {
	metrics.rendersEndedWaitingForCapacity.Inc()
}
