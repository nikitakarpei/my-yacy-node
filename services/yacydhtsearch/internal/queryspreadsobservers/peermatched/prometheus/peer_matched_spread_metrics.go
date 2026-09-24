// Package prometheus reports how long a peer matched spread took as a metric.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/budgetbuckets"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
)

type PeerMatchedSpreadMetrics struct {
	peerMatchedSpreadDurationSeconds prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *PeerMatchedSpreadMetrics {
	metrics := &PeerMatchedSpreadMetrics{
		peerMatchedSpreadDurationSeconds: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name:    "yacydhtsearch_peer_matched_spread_duration_seconds",
				Help:    "Peer matched spread duration in seconds.",
				Buckets: budgetbuckets.DurationBucketsFor(queryBudget),
			},
		),
	}
	registry.MustRegister(
		metrics.peerMatchedSpreadDurationSeconds,
	)

	return metrics
}

func (m *PeerMatchedSpreadMetrics) PeerMatchedSpreadPerformed(
	_ context.Context,
	spread peermatched.PerformedPeerMatchedSpread,
) {
	m.peerMatchedSpreadDurationSeconds.Observe(spread.TimeSpent.Seconds())
}
