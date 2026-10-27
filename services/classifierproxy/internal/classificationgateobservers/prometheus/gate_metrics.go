// Package prometheus counts the facts of the classification gate as Prometheus
// metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

var (
	slotWaitSecondsBuckets = []float64{
		0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
	}
	classificationSecondsBuckets = []float64{
		0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5,
	}
	pageSizeRanges = []pageSizeRange{
		{name: "up_to_16KiB", largestPageSize: 16 << 10},
		{name: "up_to_64KiB", largestPageSize: 64 << 10},
		{name: "up_to_256KiB", largestPageSize: 256 << 10},
		{name: "up_to_1MiB", largestPageSize: 1 << 20},
	}
)

const largerPageSizeRange = "over_1MiB"

type pageSizeRange struct {
	name            string
	largestPageSize int
}

type GateMetrics struct {
	slotWaitSeconds       prometheusclient.Histogram
	classificationSeconds *prometheusclient.HistogramVec
	panics                prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *GateMetrics {
	metrics := &GateMetrics{
		slotWaitSeconds: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "classifierproxy_classification_slot_wait_duration_seconds",
			Help:    "Time that a page waits for a free classification slot.",
			Buckets: slotWaitSecondsBuckets,
		}),
		classificationSeconds: prometheusclient.NewHistogramVec(prometheusclient.HistogramOpts{
			Name:    "classifierproxy_classification_duration_seconds",
			Help:    "Time to classify one page, by the size of the classified page.",
			Buckets: classificationSecondsBuckets,
		}, []string{"page_size"}),
		panics: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "classifierproxy_classification_panics_total",
			Help: "Classifications that panicked.",
		}),
	}
	registry.MustRegister(metrics.slotWaitSeconds, metrics.classificationSeconds, metrics.panics)
	return metrics
}

func (m *GateMetrics) SlotWaited(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	m.slotWaitSeconds.Observe(slotWait.Seconds())
}

func (m *GateMetrics) ClassificationFinished(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	pageSize int,
	classificationDuration time.Duration,
) {
	m.classificationSeconds.WithLabelValues(pageSizeRangeOf(pageSize)).
		Observe(classificationDuration.Seconds())
}

func pageSizeRangeOf(pageSize int) string {
	for _, sizeRange := range pageSizeRanges {
		if pageSize <= sizeRange.largestPageSize {
			return sizeRange.name
		}
	}
	return largerPageSizeRange
}

func (m *GateMetrics) ClassificationPanicked(context.Context, canonicalurl.CanonicalURL, any) {
	m.panics.Inc()
}
