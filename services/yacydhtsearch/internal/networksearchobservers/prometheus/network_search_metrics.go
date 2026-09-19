// Package prometheus reports network search breadth, the size of the ranking,
// and duration as metrics.
package prometheus

import (
	"context"
	"math"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
)

const (
	durationBucketRatio = 1.6
	bucketsUpToBudget   = 15
	itemBucketCeiling   = 10000.0
	itemBuckets         = 10
	peerBucketCeiling   = 128.0
	peerBuckets         = 8
)

var overBudgetShares = []float64{1.25, 1.5, 2}

type NetworkSearchMetrics struct {
	itemsRankedPerNetworkSearch  prometheusclient.Histogram
	askablePeersPerNetworkSearch prometheusclient.Histogram
	networkSearchDurationSeconds prometheusclient.Histogram
}

func New(registry prometheusclient.Registerer, queryBudget time.Duration) *NetworkSearchMetrics {
	metrics := &NetworkSearchMetrics{
		itemsRankedPerNetworkSearch: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_network_search_items_ranked",
			Help:    "Unique items in the final ranking after the ranking limit.",
			Buckets: bucketsFromNoneTo(itemBucketCeiling, itemBuckets),
		}),
		askablePeersPerNetworkSearch: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_network_search_askable_peers",
			Help:    "Peers the directory could ask when one network search started.",
			Buckets: bucketsFromNoneTo(peerBucketCeiling, peerBuckets),
		}),
		networkSearchDurationSeconds: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_network_search_duration_seconds",
			Help:    "Network search duration in seconds.",
			Buckets: networkSearchDurationBucketsFor(queryBudget),
		}),
	}
	registry.MustRegister(
		metrics.itemsRankedPerNetworkSearch,
		metrics.askablePeersPerNetworkSearch,
		metrics.networkSearchDurationSeconds,
	)

	return metrics
}

func bucketsFromNoneTo(ceiling float64, buckets int) []float64 {
	return append(
		[]float64{0},
		prometheusclient.ExponentialBucketsRange(1, ceiling, buckets)...,
	)
}

func networkSearchDurationBucketsFor(queryBudget time.Duration) []float64 {
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

func (m *NetworkSearchMetrics) NetworkSearchPerformed(
	_ context.Context,
	search networksearch.PerformedNetworkSearch,
) {
	m.itemsRankedPerNetworkSearch.Observe(float64(search.AmountOfItemsInRanking))
	m.askablePeersPerNetworkSearch.Observe(float64(search.AmountOfAskablePeers))
	m.networkSearchDurationSeconds.Observe(search.TimeSpent.Seconds())
}
