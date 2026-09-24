// Package prometheus reports network search breadth, the size of the ranking,
// and duration as metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/budgetbuckets"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
)

const (
	itemBucketCeiling = 10000.0
	itemBuckets       = 10
	peerBucketCeiling = 128.0
	peerBuckets       = 8
)

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
			Buckets: budgetbuckets.DurationBucketsFor(queryBudget),
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

func (m *NetworkSearchMetrics) NetworkSearchPerformed(
	_ context.Context,
	search networksearch.PerformedNetworkSearch,
) {
	m.itemsRankedPerNetworkSearch.Observe(float64(search.AmountOfItemsInRanking))
	m.askablePeersPerNetworkSearch.Observe(float64(search.AmountOfAskablePeers))
	m.networkSearchDurationSeconds.Observe(search.TimeSpent.Seconds())
}
