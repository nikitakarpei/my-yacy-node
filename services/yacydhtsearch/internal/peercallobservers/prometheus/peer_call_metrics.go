// Package prometheus counts peer calls by outcome and by what they asked the
// peer for, and reports what each peer call cost in time.
package prometheus

import (
	"context"
	"math"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

const (
	labelOutcome               = "outcome"
	labelAskedFor              = "asked_for"
	outcomePeerAnswered        = "answered"
	outcomePeerAnsweredNothing = "answered nothing"
	outcomePeerRefused         = "refused"
	outcomePeerUnreachable     = "unreachable"
	outcomePeerUnreadable      = "unreadable"
	durationBucketRatio        = 1.6
	bucketsUpToBudget          = 15
)

var overBudgetShares = []float64{1.25, 1.5, 2}

type PeerCallMetrics struct {
	peerCalls               *prometheusclient.CounterVec
	peerCallDurationSeconds *prometheusclient.HistogramVec
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *PeerCallMetrics {
	metrics := &PeerCallMetrics{
		peerCalls: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_peer_calls_total",
			Help: "Peer calls, by outcome and by what the peer call asked the peer for.",
		}, []string{labelOutcome, labelAskedFor}),
		peerCallDurationSeconds: prometheusclient.NewHistogramVec(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_peer_call_duration_seconds",
			Help:    "One peer call in seconds, by outcome and by what it asked for.",
			Buckets: peerCallDurationBucketsFor(queryBudget),
		}, []string{labelOutcome, labelAskedFor}),
	}
	registry.MustRegister(
		metrics.peerCalls,
		metrics.peerCallDurationSeconds,
	)

	return metrics
}

func peerCallDurationBucketsFor(queryBudget time.Duration) []float64 {
	seconds := queryBudget.Seconds()
	buckets := make([]float64, 0, bucketsUpToBudget+len(overBudgetShares))
	for step := bucketsUpToBudget - 1; step >= 0; step-- {
		buckets = append(buckets, seconds/math.Pow(durationBucketRatio, float64(step)))
	}
	for _, share := range overBudgetShares {
		buckets = append(buckets, seconds*share)
	}

	return buckets
}

func (m *PeerCallMetrics) PeerAnsweredMatchedItems(
	_ context.Context,
	_ string,
	amountOfMatchedItems int,
	spent time.Duration,
) {
	m.countAnswer(peerasks.MatchedItems, amountOfMatchedItems, spent)
}

func (m *PeerCallMetrics) PeerAnsweredURLMetadata(
	_ context.Context,
	_ string,
	amountOfDescribedDocuments int,
	spent time.Duration,
) {
	m.countAnswer(peerasks.URLMetadata, amountOfDescribedDocuments, spent)
}

func (m *PeerCallMetrics) PeerAnsweredHeldDocuments(
	_ context.Context,
	_ string,
	amountOfDocuments int,
	spent time.Duration,
) {
	m.countAnswer(peerasks.HeldDocuments, amountOfDocuments, spent)
}

func (m *PeerCallMetrics) countAnswer(
	askedFor peerasks.AskedFor,
	amountAnswered int,
	spent time.Duration,
) {
	if amountAnswered == 0 {
		m.countPeerCall(outcomePeerAnsweredNothing, askedFor, spent)

		return
	}
	m.countPeerCall(outcomePeerAnswered, askedFor, spent)
}

func (m *PeerCallMetrics) PeerRefused(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
	_ int,
	spent time.Duration,
) {
	m.countPeerCall(outcomePeerRefused, askedFor, spent)
}

func (m *PeerCallMetrics) PeerUnreachable(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
	_ error,
	spent time.Duration,
) {
	m.countPeerCall(outcomePeerUnreachable, askedFor, spent)
}

func (m *PeerCallMetrics) PeerAnswerUnreadable(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
	_ error,
	spent time.Duration,
) {
	m.countPeerCall(outcomePeerUnreadable, askedFor, spent)
}

func (m *PeerCallMetrics) countPeerCall(
	outcome string,
	askedFor peerasks.AskedFor,
	spent time.Duration,
) {
	m.peerCalls.WithLabelValues(outcome, string(askedFor)).Inc()
	m.peerCallDurationSeconds.WithLabelValues(outcome, string(askedFor)).Observe(spent.Seconds())
}
