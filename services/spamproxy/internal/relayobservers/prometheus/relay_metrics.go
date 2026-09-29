// Package prometheus counts the facts of the relay as Prometheus metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/pagerelay"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/spamassessment"
)

var (
	scoreBuckets             = prometheusclient.LinearBuckets(0.1, 0.1, 10)
	assessmentSecondsBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5}
)

type RelayMetrics struct {
	verdicts          *prometheusclient.CounterVec
	scores            prometheusclient.Histogram
	assessmentSeconds prometheusclient.Histogram
	nonPages          *prometheusclient.CounterVec
	refusals          *prometheusclient.CounterVec
	relayFailures     prometheusclient.Counter
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
			Name:    "spamproxy_assessment_seconds",
			Help:    "Time to assess one page.",
			Buckets: assessmentSecondsBuckets,
		}),
		nonPages: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_non_pages_total",
			Help: "Answers relayed without a verdict because they are not pages.",
		}, []string{"reason"}),
		refusals: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_refusals_total",
			Help: "Answers refused with an error status.",
		}, []string{"reason"}),
		relayFailures: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "spamproxy_relay_failures_total",
			Help: "Relays that failed.",
		}),
	}
	registry.MustRegister(
		metrics.verdicts,
		metrics.scores,
		metrics.assessmentSeconds,
		metrics.nonPages,
		metrics.refusals,
		metrics.relayFailures,
	)
	return metrics
}

func (m *RelayMetrics) PageAssessed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	assessment spamassessment.Assessment,
	assessmentDuration time.Duration,
) {
	m.verdicts.WithLabelValues(assessment.Verdict()).Inc()
	m.scores.Observe(assessment.Score)
	m.assessmentSeconds.Observe(assessmentDuration.Seconds())
}

func (m *RelayMetrics) NonPageRelayed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason pagerelay.NonPageReason,
) {
	m.nonPages.WithLabelValues(string(reason)).Inc()
}

func (m *RelayMetrics) AnswerRefused(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason pagerelay.RefusalReason,
) {
	m.refusals.WithLabelValues(string(reason)).Inc()
}

func (m *RelayMetrics) RelayFailed(context.Context, canonicalurl.CanonicalURL, error) {
	m.relayFailures.Inc()
}
