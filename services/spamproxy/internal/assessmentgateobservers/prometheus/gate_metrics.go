// Package prometheus counts the facts of the assessment gate as Prometheus
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
	assessmentSecondsBuckets = []float64{
		0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5,
	}
)

type GateMetrics struct {
	slotWaitSeconds   prometheusclient.Histogram
	assessmentSeconds prometheusclient.Histogram
	panics            prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *GateMetrics {
	metrics := &GateMetrics{
		slotWaitSeconds: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "spamproxy_assessment_slot_wait_duration_seconds",
			Help:    "Time that a page waits for a free assessment slot.",
			Buckets: slotWaitSecondsBuckets,
		}),
		assessmentSeconds: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "spamproxy_assessment_duration_seconds",
			Help:    "Time to assess one page.",
			Buckets: assessmentSecondsBuckets,
		}),
		panics: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "spamproxy_assessment_panics_total",
			Help: "Assessments that panicked.",
		}),
	}
	registry.MustRegister(metrics.slotWaitSeconds, metrics.assessmentSeconds, metrics.panics)
	return metrics
}

func (m *GateMetrics) SlotWaited(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	m.slotWaitSeconds.Observe(slotWait.Seconds())
}

func (m *GateMetrics) AssessmentFinished(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	assessmentDuration time.Duration,
) {
	m.assessmentSeconds.Observe(assessmentDuration.Seconds())
}

func (m *GateMetrics) AssessmentPanicked(context.Context, canonicalurl.CanonicalURL, any) {
	m.panics.Inc()
}
