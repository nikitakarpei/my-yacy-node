// Package prometheus counts the facts of the relay as Prometheus metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelay"
)

var (
	scoreBuckets             = prometheusclient.LinearBuckets(0.1, 0.1, 10)
	assessmentSecondsBuckets = []float64{
		0.001,
		0.0025,
		0.005,
		0.01,
		0.025,
		0.05,
		0.1,
		0.25,
		0.5,
		1,
		2.5,
	}
)

type RelayMetrics struct {
	verdicts              *prometheusclient.CounterVec
	scores                prometheusclient.Histogram
	assessmentSeconds     prometheusclient.Histogram
	skippedAssessments    *prometheusclient.CounterVec
	refusals              *prometheusclient.CounterVec
	answerReadingFailures prometheusclient.Counter
	cutShortReplies       prometheusclient.Counter
	departedClients       prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *RelayMetrics {
	metrics := &RelayMetrics{
		verdicts: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_verdicts_total",
			Help: "Assessed pages by verdict.",
		}, []string{"verdict"}),
		scores: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "spamproxy_scores",
			Help:    "Scores of the assessed pages.",
			Buckets: scoreBuckets,
		}),
		assessmentSeconds: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "spamproxy_assessment_duration_seconds",
			Help:    "Time to assess one page.",
			Buckets: assessmentSecondsBuckets,
		}),
		skippedAssessments: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_skipped_assessments_total",
			Help: "Answers relayed without an assessment.",
		}, []string{"reason"}),
		refusals: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_refusals_total",
			Help: "Answers refused with an error status.",
		}, []string{"reason"}),
		answerReadingFailures: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "spamproxy_answer_reading_failures_total",
			Help: "Answers of the egress proxy that failed before the response headers.",
		}),
		cutShortReplies: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "spamproxy_cut_short_replies_total",
			Help: "Replies that ended before the whole body.",
		}),
		departedClients: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "spamproxy_departed_clients_total",
			Help: "Clients that left before the reply ended.",
		}),
	}
	registry.MustRegister(
		metrics.verdicts,
		metrics.scores,
		metrics.assessmentSeconds,
		metrics.skippedAssessments,
		metrics.refusals,
		metrics.answerReadingFailures,
		metrics.cutShortReplies,
		metrics.departedClients,
	)
	return metrics
}

func (m *RelayMetrics) PageAssessed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	assessment spamassessment.Assessment,
	assessmentDuration time.Duration,
) {
	m.verdicts.WithLabelValues(assessment.Verdict().String()).Inc()
	m.scores.Observe(assessment.Score)
	m.assessmentSeconds.Observe(assessmentDuration.Seconds())
}

func (m *RelayMetrics) AssessmentSkipped(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason requestrelay.SkipReason,
) {
	m.skippedAssessments.WithLabelValues(string(reason)).Inc()
}

func (m *RelayMetrics) RequestRefused(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason requestrelay.RefusalReason,
) {
	m.refusals.WithLabelValues(string(reason)).Inc()
}

func (m *RelayMetrics) AnswerReadingFailed(context.Context, canonicalurl.CanonicalURL, error) {
	m.answerReadingFailures.Inc()
}

func (m *RelayMetrics) ReplyCutShort(context.Context, canonicalurl.CanonicalURL, error) {
	m.cutShortReplies.Inc()
}

func (m *RelayMetrics) ClientLeft(context.Context, canonicalurl.CanonicalURL) {
	m.departedClients.Inc()
}
