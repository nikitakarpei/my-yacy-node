// Package prometheus reports the outcome of each network search, its breadth,
// the size of its ranking, and its duration as metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/budgetbuckets"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	labelOutcome       = "outcome"
	outcomePeersAsked  = "peers asked"
	outcomeNoPeerToAsk = "no peer to ask"
	itemBucketCeiling  = 10000.0
	itemBuckets        = 10
	peerBucketCeiling  = 128.0
	peerBuckets        = 8
)

type NetworkSearchMetrics struct {
	searchesThatAskedPeers       prometheusclient.Counter
	searchesWithNoPeerToAsk      prometheusclient.Counter
	itemsRankedPerNetworkSearch  prometheusclient.Histogram
	askablePeersPerNetworkSearch prometheusclient.Histogram
	networkSearchDurationSeconds prometheusclient.Histogram
}

func New(registry prometheusclient.Registerer, queryBudget time.Duration) *NetworkSearchMetrics {
	searches := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_network_searches_total",
		Help: "Network searches, by the outcome each search reached.",
	}, []string{labelOutcome})
	metrics := &NetworkSearchMetrics{
		searchesThatAskedPeers:  searches.WithLabelValues(outcomePeersAsked),
		searchesWithNoPeerToAsk: searches.WithLabelValues(outcomeNoPeerToAsk),
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
		searches,
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
	m.searchesThatAskedPeers.Inc()
	m.itemsRankedPerNetworkSearch.Observe(float64(search.AmountOfItemsInRanking))
	m.askablePeersPerNetworkSearch.Observe(float64(search.AmountOfAskablePeers))
	m.networkSearchDurationSeconds.Observe(search.TimeSpent.Seconds())
}

func (m *NetworkSearchMetrics) QueryReachedNoPeer(context.Context, searchquery.Query) {
	m.searchesWithNoPeerToAsk.Inc()
}
