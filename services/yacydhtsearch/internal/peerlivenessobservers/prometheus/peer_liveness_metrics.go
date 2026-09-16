// Package prometheus reports why the probes of peer addresses found no peer.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

const (
	labelFailure        = "failure"
	failureProbeUnbuilt = "probeUnbuilt"
	failureNoAnswer     = "noAnswer"
	failureRefused      = "refused"
	failureAnswerUnread = "answerUnread"
	failureNoRWICount   = "noRWICount"
)

type PeerLivenessMetrics struct {
	probeFailures *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *PeerLivenessMetrics {
	metrics := &PeerLivenessMetrics{
		probeFailures: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_probe_failures_total",
			Help: "Probes of a peer address that found no peer, by failure.",
		}, []string{labelFailure}),
	}
	registry.MustRegister(metrics.probeFailures)

	return metrics
}

func (m *PeerLivenessMetrics) ProbeCouldNotBeBuilt(context.Context, string, error) {
	m.probeFailures.WithLabelValues(failureProbeUnbuilt).Inc()
}

func (m *PeerLivenessMetrics) PeerDidNotAnswerTheProbe(context.Context, string, error) {
	m.probeFailures.WithLabelValues(failureNoAnswer).Inc()
}

func (m *PeerLivenessMetrics) PeerRefusedTheProbe(context.Context, string, int) {
	m.probeFailures.WithLabelValues(failureRefused).Inc()
}

func (m *PeerLivenessMetrics) ProbeAnswerCouldNotBeRead(context.Context, string, error) {
	m.probeFailures.WithLabelValues(failureAnswerUnread).Inc()
}

func (m *PeerLivenessMetrics) ProbeAnswerCarriedNoRWICount(context.Context, string, error) {
	m.probeFailures.WithLabelValues(failureNoRWICount).Inc()
}
