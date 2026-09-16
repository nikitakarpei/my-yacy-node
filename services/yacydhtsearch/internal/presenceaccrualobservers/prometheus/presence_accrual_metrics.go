// Package prometheus reports how much presence the peers of this deployment
// have earned and how many of them presence is held for.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PresenceAccrualMetrics struct {
	firstAnswers   prometheusclient.Counter
	earnedPresence prometheusclient.Histogram
	observedPeers  prometheusclient.Gauge
}

func New(registry prometheusclient.Registerer) *PresenceAccrualMetrics {
	metrics := &PresenceAccrualMetrics{
		firstAnswers: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_peer_presence_first_answers_total",
			Help: "Peers at an address that answered for the first time.",
		}),
		earnedPresence: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_peer_presence_earned_seconds",
			Help:    "Presence a peer holds after an answer added to it.",
			Buckets: prometheusclient.ExponentialBuckets(60, 4, 8),
		}),
		observedPeers: prometheusclient.NewGauge(prometheusclient.GaugeOpts{
			Name: "yacydhtsearch_peer_presence_observed_peers",
			Help: "Peers at an address presence is held for.",
		}),
	}
	registry.MustRegister(metrics.firstAnswers, metrics.earnedPresence, metrics.observedPeers)

	return metrics
}

func (m *PresenceAccrualMetrics) PeerAnsweredForTheFirstTime(
	context.Context,
	yacymodel.Hash,
	string,
) {
	m.firstAnswers.Inc()
}

func (m *PresenceAccrualMetrics) PeerEarnedPresence(
	_ context.Context,
	_ yacymodel.Hash,
	presence time.Duration,
) {
	m.earnedPresence.Observe(presence.Seconds())
}

func (m *PresenceAccrualMetrics) PeersObserved(_ context.Context, amountOfObservedPeers int) {
	m.observedPeers.Set(float64(amountOfObservedPeers))
}
