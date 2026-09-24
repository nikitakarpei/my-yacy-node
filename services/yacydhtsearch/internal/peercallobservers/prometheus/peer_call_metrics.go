// Package prometheus counts peer calls by outcome and by what they asked the
// peer for, and reports what each peer call cost in time.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/budgetbuckets"
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
	outcomePeerCallCancelled   = "cancelled"
)

type PeerCallMetrics struct {
	peerCallsPerAskedFor     map[peerasks.AskedFor]peerCallsOfOneAsk
	peerCallsWaitingForASlot prometheusclient.Gauge
}

type peerCallsOfOneAsk struct {
	answered        peerCallsOfOneOutcome
	answeredNothing peerCallsOfOneOutcome
	refused         peerCallsOfOneOutcome
	unreachable     peerCallsOfOneOutcome
	unreadable      peerCallsOfOneOutcome
	cancelled       peerCallsOfOneOutcome
}

type peerCallsOfOneOutcome struct {
	peerCalls               prometheusclient.Counter
	peerCallDurationSeconds prometheusclient.Observer
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *PeerCallMetrics {
	peerCalls := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_peer_calls_total",
		Help: "Peer calls, by outcome and by what the peer call asked the peer for.",
	}, []string{labelOutcome, labelAskedFor})
	peerCallDurationSeconds := prometheusclient.NewHistogramVec(prometheusclient.HistogramOpts{
		Name:    "yacydhtsearch_peer_call_duration_seconds",
		Help:    "One peer call in seconds, by outcome and by what it asked for.",
		Buckets: budgetbuckets.DurationBucketsFor(queryBudget),
	}, []string{labelOutcome, labelAskedFor})
	peerCallsWaitingForASlot := prometheusclient.NewGauge(prometheusclient.GaugeOpts{
		Name: "yacydhtsearch_peer_calls_waiting_for_a_slot",
		Help: "How many peer calls wait for an in-flight slot.",
	})
	registry.MustRegister(peerCalls, peerCallDurationSeconds, peerCallsWaitingForASlot)

	//exhaustive:enforce
	peerCallsPerAskedFor := map[peerasks.AskedFor]peerCallsOfOneAsk{
		peerasks.SearchDocuments: peerCallsOfOneAskFrom(
			peerCalls, peerCallDurationSeconds, peerasks.SearchDocuments,
		),
		peerasks.URLMetadata: peerCallsOfOneAskFrom(
			peerCalls, peerCallDurationSeconds, peerasks.URLMetadata,
		),
	}

	return &PeerCallMetrics{
		peerCallsPerAskedFor:     peerCallsPerAskedFor,
		peerCallsWaitingForASlot: peerCallsWaitingForASlot,
	}
}

func peerCallsOfOneAskFrom(
	peerCalls *prometheusclient.CounterVec,
	peerCallDurationSeconds *prometheusclient.HistogramVec,
	askedFor peerasks.AskedFor,
) peerCallsOfOneAsk {
	return peerCallsOfOneAsk{
		answered: peerCallsOfOneOutcomeFrom(
			peerCalls, peerCallDurationSeconds, askedFor, outcomePeerAnswered,
		),
		answeredNothing: peerCallsOfOneOutcomeFrom(
			peerCalls, peerCallDurationSeconds, askedFor, outcomePeerAnsweredNothing,
		),
		refused: peerCallsOfOneOutcomeFrom(
			peerCalls, peerCallDurationSeconds, askedFor, outcomePeerRefused,
		),
		unreachable: peerCallsOfOneOutcomeFrom(
			peerCalls, peerCallDurationSeconds, askedFor, outcomePeerUnreachable,
		),
		unreadable: peerCallsOfOneOutcomeFrom(
			peerCalls, peerCallDurationSeconds, askedFor, outcomePeerUnreadable,
		),
		cancelled: peerCallsOfOneOutcomeFrom(
			peerCalls, peerCallDurationSeconds, askedFor, outcomePeerCallCancelled,
		),
	}
}

func peerCallsOfOneOutcomeFrom(
	peerCalls *prometheusclient.CounterVec,
	peerCallDurationSeconds *prometheusclient.HistogramVec,
	askedFor peerasks.AskedFor,
	outcome string,
) peerCallsOfOneOutcome {
	return peerCallsOfOneOutcome{
		peerCalls:               peerCalls.WithLabelValues(outcome, string(askedFor)),
		peerCallDurationSeconds: peerCallDurationSeconds.WithLabelValues(outcome, string(askedFor)),
	}
}

func (m *PeerCallMetrics) PeerCallWaitsForASlot(
	_ context.Context,
	_ string,
	_ peerasks.AskedFor,
) {
	m.peerCallsWaitingForASlot.Inc()
}

func (m *PeerCallMetrics) PeerCallTookASlot(
	_ context.Context,
	_ string,
	_ peerasks.AskedFor,
	_ time.Duration,
) {
	m.peerCallsWaitingForASlot.Dec()
}

func (m *PeerCallMetrics) PeerAnsweredURLMetadata(
	_ context.Context,
	_ string,
	amountOfDescribedDocuments int,
	spent time.Duration,
) {
	m.peerCallsPerAskedFor[peerasks.URLMetadata].countAnswer(amountOfDescribedDocuments, spent)
}

func (m *PeerCallMetrics) PeerSearchedDocuments(
	_ context.Context,
	_ string,
	amountOfDocumentsInTheAbstract int,
	amountOfMatchedDocuments int,
	spent time.Duration,
) {
	m.peerCallsPerAskedFor[peerasks.SearchDocuments].countAnswer(
		amountOfDocumentsInTheAbstract+amountOfMatchedDocuments, spent,
	)
}

func (calls peerCallsOfOneAsk) countAnswer(amountAnswered int, spent time.Duration) {
	if amountAnswered == 0 {
		calls.answeredNothing.count(spent)

		return
	}
	calls.answered.count(spent)
}

func (calls peerCallsOfOneOutcome) count(spent time.Duration) {
	calls.peerCalls.Inc()
	calls.peerCallDurationSeconds.Observe(spent.Seconds())
}

func (m *PeerCallMetrics) PeerRefused(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
	_ int,
	spent time.Duration,
) {
	m.peerCallsPerAskedFor[askedFor].refused.count(spent)
}

func (m *PeerCallMetrics) PeerUnreachable(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
	_ error,
	spent time.Duration,
) {
	m.peerCallsPerAskedFor[askedFor].unreachable.count(spent)
}

func (m *PeerCallMetrics) PeerAnswerUnreadable(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
	_ error,
	spent time.Duration,
) {
	m.peerCallsPerAskedFor[askedFor].unreadable.count(spent)
}

func (m *PeerCallMetrics) PeerCallCancelled(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
	spent time.Duration,
) {
	m.peerCallsPerAskedFor[askedFor].cancelled.count(spent)
}
