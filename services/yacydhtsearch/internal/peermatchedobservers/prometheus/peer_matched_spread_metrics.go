// Package prometheus reports how many of the peers a peer matched spread asked
// answered it, and how long the spread took, as metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
)

const (
	durationBuckets = 12
	budgetShare     = 1024
	ratioBuckets    = 11
)

type PeerMatchedSpreadMetrics struct {
	answeringPeersRatio              prometheusclient.Histogram
	peerMatchedSpreadDurationSeconds prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *PeerMatchedSpreadMetrics {
	metrics := &PeerMatchedSpreadMetrics{
		answeringPeersRatio: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_peer_matched_spread_answering_peers_ratio",
			Help:    "Share of asked peers that answered during a peer matched spread.",
			Buckets: prometheusclient.LinearBuckets(0, 0.1, ratioBuckets),
		}),
		peerMatchedSpreadDurationSeconds: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_peer_matched_spread_duration_seconds",
				Help: "Peer matched spread duration in seconds.",
				Buckets: prometheusclient.ExponentialBucketsRange(
					queryBudget.Seconds()/budgetShare,
					queryBudget.Seconds()*2,
					durationBuckets,
				),
			},
		),
	}
	registry.MustRegister(
		metrics.answeringPeersRatio,
		metrics.peerMatchedSpreadDurationSeconds,
	)

	return metrics
}

func (m *PeerMatchedSpreadMetrics) PeerMatchedSpreadPerformed(
	_ context.Context,
	spread peermatched.PerformedPeerMatchedSpread,
) {
	m.peerMatchedSpreadDurationSeconds.Observe(spread.TimeSpent.Seconds())
	if spread.AmountOfPeersAsked != 0 {
		m.answeringPeersRatio.Observe(
			float64(spread.AmountOfPeersThatAnswered) / float64(spread.AmountOfPeersAsked),
		)
	}
}
