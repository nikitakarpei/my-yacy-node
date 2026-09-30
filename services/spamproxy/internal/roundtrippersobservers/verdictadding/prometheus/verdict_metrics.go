// Package prometheus counts the facts of adding verdicts to pages as
// Prometheus metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/verdictadding"
)

var (
	scoreBuckets           = prometheusclient.LinearBuckets(0.1, 0.1, 10)
	slotWaitSecondsBuckets = []float64{
		0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
	}
)

type VerdictMetrics struct {
	verdicts               *prometheusclient.CounterVec
	scores                 prometheusclient.Histogram
	readingSlotWaitSeconds prometheusclient.Histogram
	skippedAssessments     *prometheusclient.CounterVec
	refusedPages           *prometheusclient.CounterVec
	bodyPrefixReadFailures prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *VerdictMetrics {
	metrics := &VerdictMetrics{
		verdicts: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_verdicts_total",
			Help: "Assessed pages by verdict.",
		}, []string{"verdict"}),
		scores: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "spamproxy_scores",
			Help:    "Scores of the assessed pages.",
			Buckets: scoreBuckets,
		}),
		readingSlotWaitSeconds: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "spamproxy_reading_slot_wait_duration_seconds",
			Help:    "Time that a page waits for a place among the pages read at once.",
			Buckets: slotWaitSecondsBuckets,
		}),
		skippedAssessments: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_skipped_assessments_total",
			Help: "Upstream responses relayed without an assessment.",
		}, []string{"reason"}),
		refusedPages: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_refused_pages_total",
			Help: "Pages that the proxy answers with its own error status, by reason.",
		}, []string{"reason"}),
		bodyPrefixReadFailures: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "spamproxy_body_prefix_read_failures_total",
			Help: "Pages whose body failed before the part that the proxy assesses.",
		}),
	}
	registry.MustRegister(
		metrics.verdicts,
		metrics.scores,
		metrics.readingSlotWaitSeconds,
		metrics.skippedAssessments,
		metrics.refusedPages,
		metrics.bodyPrefixReadFailures,
	)
	return metrics
}

func (m *VerdictMetrics) PageAssessed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	assessment spamassessment.Assessment,
) {
	m.verdicts.WithLabelValues(assessment.Verdict().String()).Inc()
	m.scores.Observe(assessment.Score)
}

func (m *VerdictMetrics) ReadingSlotWaited(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	m.readingSlotWaitSeconds.Observe(slotWait.Seconds())
}

func (m *VerdictMetrics) AssessmentSkipped(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason verdictadding.SkipReason,
) {
	m.skippedAssessments.WithLabelValues(string(reason)).Inc()
}

func (m *VerdictMetrics) PageRefused(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason verdictadding.RefusalReason,
) {
	m.refusedPages.WithLabelValues(string(reason)).Inc()
}

func (m *VerdictMetrics) BodyPrefixReadFailed(context.Context, canonicalurl.CanonicalURL, error) {
	m.bodyPrefixReadFailures.Inc()
}
