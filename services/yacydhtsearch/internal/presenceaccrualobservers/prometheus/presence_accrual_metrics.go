// Package prometheus reports how much presence the peers of this deployment
// have earned.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PresenceAccrualMetrics struct {
	earnedPresence prometheusclient.Histogram
}

func New(registry prometheusclient.Registerer) *PresenceAccrualMetrics {
	metrics := &PresenceAccrualMetrics{
		earnedPresence: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_peer_presence_earned_seconds",
			Help:    "Presence a peer holds after an answer added to it.",
			Buckets: prometheusclient.ExponentialBuckets(60, 4, 8),
		}),
	}
	registry.MustRegister(metrics.earnedPresence)

	return metrics
}

func (*PresenceAccrualMetrics) PeerAnsweredForTheFirstTime(
	context.Context,
	yacymodel.Hash,
	string,
) {
}

func (m *PresenceAccrualMetrics) PeerEarnedPresence(
	_ context.Context,
	_ yacymodel.Hash,
	presence time.Duration,
) {
	m.earnedPresence.Observe(presence.Seconds())
}
