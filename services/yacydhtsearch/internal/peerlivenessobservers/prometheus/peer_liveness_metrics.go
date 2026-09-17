// Package prometheus reports why the probes of peer addresses found no peer.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

const (
	labelFailure            = "failure"
	failureUnusableAddress  = "unusableAddress"
	failureNoAnswer         = "noAnswer"
	failureRefused          = "refused"
	failureAnswerIncomplete = "answerIncomplete"
	failureNoRWICount       = "noRWICount"
)

type PeerLivenessMetrics struct {
	probesOfUnusableAddresses    prometheusclient.Counter
	probesWithNoAnswer           prometheusclient.Counter
	probesRefused                prometheusclient.Counter
	probesWithAnIncompleteAnswer prometheusclient.Counter
	probesWithNoRWICount         prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *PeerLivenessMetrics {
	probeFailures := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_probe_failures_total",
		Help: "Probes of a peer address that found no peer, by failure.",
	}, []string{labelFailure})
	registry.MustRegister(probeFailures)

	return &PeerLivenessMetrics{
		probesOfUnusableAddresses:    probeFailures.WithLabelValues(failureUnusableAddress),
		probesWithNoAnswer:           probeFailures.WithLabelValues(failureNoAnswer),
		probesRefused:                probeFailures.WithLabelValues(failureRefused),
		probesWithAnIncompleteAnswer: probeFailures.WithLabelValues(failureAnswerIncomplete),
		probesWithNoRWICount:         probeFailures.WithLabelValues(failureNoRWICount),
	}
}

func (m *PeerLivenessMetrics) ProbeCouldNotBeBuilt(context.Context, string, error) {
	m.probesOfUnusableAddresses.Inc()
}

func (m *PeerLivenessMetrics) PeerDidNotAnswerTheProbe(context.Context, string, error) {
	m.probesWithNoAnswer.Inc()
}

func (m *PeerLivenessMetrics) PeerRefusedTheProbe(context.Context, string, int) {
	m.probesRefused.Inc()
}

func (m *PeerLivenessMetrics) ProbeAnswerCouldNotBeRead(context.Context, string, error) {
	m.probesWithAnIncompleteAnswer.Inc()
}

func (m *PeerLivenessMetrics) ProbeAnswerCarriedNoRWICount(context.Context, string, error) {
	m.probesWithNoRWICount.Inc()
}
