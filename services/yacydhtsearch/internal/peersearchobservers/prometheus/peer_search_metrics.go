// Package prometheus counts calls to peers by outcome, and reports what each
// call cost in time and how many items each answer carried.
package prometheus

import (
	"context"
	"math"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

const (
	labelOutcome           = "outcome"
	outcomePeerAnswered    = "answered"
	outcomePeerRefused     = "refused"
	outcomePeerUnreachable = "unreachable"
	outcomePeerUnreadable  = "unreadable"
	durationBucketRatio    = 1.6
	bucketsUpToBudget      = 15
)

var overBudgetShares = []float64{1.25, 1.5, 2}

type PeerSearchMetrics struct {
	peerCalls           *prometheusclient.CounterVec
	callDurationSeconds *prometheusclient.HistogramVec
	answerItems         prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	peerCallBudget time.Duration,
	peerResultCeiling int,
) *PeerSearchMetrics {
	metrics := &PeerSearchMetrics{
		peerCalls: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_peer_calls_total",
			Help: "Calls to peers, by outcome.",
		}, []string{labelOutcome}),
		callDurationSeconds: prometheusclient.NewHistogramVec(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_peer_call_duration_seconds",
			Help:    "One call to one peer in seconds, by outcome.",
			Buckets: peerCallDurationBucketsFor(peerCallBudget),
		}, []string{labelOutcome}),
		answerItems: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_peer_answer_items",
			Help:    "Items one peer returned for one search.",
			Buckets: prometheusclient.LinearBuckets(0, 1, peerResultCeiling+1),
		}),
	}
	registry.MustRegister(metrics.peerCalls, metrics.callDurationSeconds, metrics.answerItems)

	return metrics
}

func peerCallDurationBucketsFor(peerCallBudget time.Duration) []float64 {
	seconds := peerCallBudget.Seconds()
	buckets := make([]float64, 0, bucketsUpToBudget+len(overBudgetShares))
	for step := bucketsUpToBudget - 1; step >= 0; step-- {
		buckets = append(buckets, seconds/math.Pow(durationBucketRatio, float64(step)))
	}
	for _, share := range overBudgetShares {
		buckets = append(buckets, seconds*share)
	}

	return buckets
}

func (m *PeerSearchMetrics) PeerAnswered(
	_ context.Context,
	_ string,
	answeredItems []searchresult.Item,
	spent time.Duration,
) {
	m.countCall(outcomePeerAnswered, spent)
	m.answerItems.Observe(float64(len(answeredItems)))
}

func (m *PeerSearchMetrics) PeerRefused(_ context.Context, _ string, _ int, spent time.Duration) {
	m.countCall(outcomePeerRefused, spent)
}

func (m *PeerSearchMetrics) PeerUnreachable(
	_ context.Context,
	_ string,
	_ error,
	spent time.Duration,
) {
	m.countCall(outcomePeerUnreachable, spent)
}

func (m *PeerSearchMetrics) PeerAnswerUnreadable(
	_ context.Context,
	_ string,
	_ error,
	spent time.Duration,
) {
	m.countCall(outcomePeerUnreadable, spent)
}

func (m *PeerSearchMetrics) countCall(outcome string, spent time.Duration) {
	m.peerCalls.WithLabelValues(outcome).Inc()
	m.callDurationSeconds.WithLabelValues(outcome).Observe(spent.Seconds())
}
