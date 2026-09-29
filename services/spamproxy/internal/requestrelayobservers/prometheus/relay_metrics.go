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
	scoreBuckets           = prometheusclient.LinearBuckets(0.1, 0.1, 10)
	slotWaitSecondsBuckets = []float64{
		0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
	}
)

type RelayMetrics struct {
	verdicts               *prometheusclient.CounterVec
	scores                 prometheusclient.Histogram
	readingSlotWaitSeconds prometheusclient.Histogram
	skippedAssessments     *prometheusclient.CounterVec
	refusedRequests        *prometheusclient.CounterVec
	upstreamFailures       *prometheusclient.CounterVec
	incompleteResponses    *prometheusclient.CounterVec
	closedRequests         prometheusclient.Counter
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
		readingSlotWaitSeconds: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "spamproxy_reading_slot_wait_duration_seconds",
			Help:    "Time that a page waits for a place among the pages read at once.",
			Buckets: slotWaitSecondsBuckets,
		}),
		skippedAssessments: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_skipped_assessments_total",
			Help: "Upstream responses relayed without an assessment.",
		}, []string{"reason"}),
		refusedRequests: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_refused_requests_total",
			Help: "Requests that the proxy answers with its own error status.",
		}, []string{"reason"}),
		upstreamFailures: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_upstream_response_failures_total",
			Help: "Upstream responses that failed before the response headers, by cause.",
		}, []string{"cause"}),
		incompleteResponses: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_incomplete_responses_total",
			Help: "Responses that ended before the whole body, by cause.",
		}, []string{"cause"}),
		closedRequests: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "spamproxy_client_closed_requests_total",
			Help: "Requests that the client closed before the response headers.",
		}),
	}
	registry.MustRegister(
		metrics.verdicts,
		metrics.scores,
		metrics.readingSlotWaitSeconds,
		metrics.skippedAssessments,
		metrics.refusedRequests,
		metrics.upstreamFailures,
		metrics.incompleteResponses,
		metrics.closedRequests,
	)
	return metrics
}

func (m *RelayMetrics) PageAssessed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	assessment spamassessment.Assessment,
) {
	m.verdicts.WithLabelValues(assessment.Verdict().String()).Inc()
	m.scores.Observe(assessment.Score)
}

func (m *RelayMetrics) ReadingSlotWaited(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	m.readingSlotWaitSeconds.Observe(slotWait.Seconds())
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
	m.refusedRequests.WithLabelValues(string(reason)).Inc()
}

func (m *RelayMetrics) UpstreamResponseFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	failure requestrelay.UpstreamResponseFailure,
	_ error,
) {
	m.upstreamFailures.WithLabelValues(string(failure)).Inc()
}

func (m *RelayMetrics) ResponseLeftIncomplete(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	incompleteResponseCause requestrelay.IncompleteResponseCause,
	_ error,
) {
	m.incompleteResponses.WithLabelValues(string(incompleteResponseCause)).Inc()
}

func (m *RelayMetrics) ClientClosedRequest(context.Context, canonicalurl.CanonicalURL) {
	m.closedRequests.Inc()
}
