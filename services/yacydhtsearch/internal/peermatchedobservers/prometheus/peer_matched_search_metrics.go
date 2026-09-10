// Package prometheus reports how many of the peers a peer matched search asked
// answered it, and how long the search took, as metrics.
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

type PeerMatchedSearchMetrics struct {
	answeringPeersRatio              prometheusclient.Histogram
	peerMatchedSearchDurationSeconds prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *PeerMatchedSearchMetrics {
	metrics := &PeerMatchedSearchMetrics{
		answeringPeersRatio: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_peer_matched_search_answering_peers_ratio",
			Help:    "Share of asked peers that answered during a peer matched search.",
			Buckets: prometheusclient.LinearBuckets(0, 0.1, ratioBuckets),
		}),
		peerMatchedSearchDurationSeconds: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_peer_matched_search_duration_seconds",
				Help: "Peer matched search duration in seconds.",
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
		metrics.peerMatchedSearchDurationSeconds,
	)

	return metrics
}

func (m *PeerMatchedSearchMetrics) PeerMatchedSearchPerformed(
	_ context.Context,
	search peermatched.PerformedPeerMatchedSearch,
) {
	m.peerMatchedSearchDurationSeconds.Observe(search.TimeSpent.Seconds())
	if search.AmountOfAskedPeers != 0 {
		m.answeringPeersRatio.Observe(
			float64(search.AmountOfPeersThatAnswered) / float64(search.AmountOfAskedPeers),
		)
	}
}
