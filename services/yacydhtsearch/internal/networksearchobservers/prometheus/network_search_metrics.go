// Package prometheus reports how many peers a network search reached, and its
// answering peers, peers that sent items, overlap and duration, as metrics. It
// also counts the searches that reached no peer, because the query holds no
// indexed word.
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
)

var overBudgetShares = []float64{1.25, 1.5, 2}

type NetworkSearchMetrics struct {
	networkSearchesPerformed     prometheusclient.Counter
	searchesWithoutIndexedTerm   prometheusclient.Counter
	peersAskedPerNetworkSearch   prometheusclient.Histogram
	answeringPeersRatio          prometheusclient.Histogram
	peersThatSentItemsRatio      prometheusclient.Histogram
	networkSearchOverlap         prometheusclient.Histogram
	networkSearchDurationSeconds prometheusclient.Histogram
}

func New(registry prometheusclient.Registerer, queryBudget time.Duration) *NetworkSearchMetrics {
	metrics := &NetworkSearchMetrics{
		networkSearchesPerformed: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_network_searches_performed_total",
			Help: "Network searches that asked at least one peer.",
		}),
		searchesWithoutIndexedTerm: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_searches_without_indexed_term_total",
			Help: "Searches no peer can answer, because no query word is long enough to be indexed.",
		}),
		peersAskedPerNetworkSearch: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_network_search_peers_asked",
			Help:    "Peers asked for one network search. Zero means the directory held none.",
			Buckets: prometheusclient.ExponentialBucketsRange(1, 128, 8),
		}),
		answeringPeersRatio: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_network_search_answering_peers_ratio",
			Help:    "Share of asked peers that replied.",
			Buckets: prometheusclient.LinearBuckets(0, 0.1, 11),
		}),
		peersThatSentItemsRatio: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_network_search_peers_that_sent_items_ratio",
			Help:    "Share of replying peers that sent at least one item.",
			Buckets: prometheusclient.LinearBuckets(0, 0.1, 11),
		}),
		networkSearchOverlap: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_network_search_overlap_ratio",
			Help:    "Share of answered items that repeat an address another peer answered.",
			Buckets: prometheusclient.LinearBuckets(0, 0.1, 11),
		}),
		networkSearchDurationSeconds: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_network_search_duration_seconds",
			Help:    "Network search duration in seconds.",
			Buckets: networkSearchDurationBucketsFor(queryBudget),
		}),
	}
	registry.MustRegister(
		metrics.networkSearchesPerformed,
		metrics.searchesWithoutIndexedTerm,
		metrics.peersAskedPerNetworkSearch,
		metrics.answeringPeersRatio,
		metrics.peersThatSentItemsRatio,
		metrics.networkSearchOverlap,
		metrics.networkSearchDurationSeconds,
	)

	return metrics
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
	m.networkSearchesPerformed.Inc()
	m.peersAskedPerNetworkSearch.Observe(float64(search.AmountOfAskedPeers))
	m.networkSearchDurationSeconds.Observe(search.TimeSpent.Seconds())
	m.answeringPeersRatio.Observe(
		float64(search.AmountOfAnsweringPeers) / float64(search.AmountOfAskedPeers),
	)
	m.observePeersThatSentItemsRatio(search)
	m.observeOverlap(search)
}

func (m *NetworkSearchMetrics) observePeersThatSentItemsRatio(
	search networksearch.PerformedNetworkSearch,
) {
	if search.AmountOfAnsweringPeers == 0 {
		return
	}

	m.peersThatSentItemsRatio.Observe(
		float64(search.AmountOfPeersThatSentItems) /
			float64(search.AmountOfAnsweringPeers),
	)
}

func (m *NetworkSearchMetrics) observeOverlap(search networksearch.PerformedNetworkSearch) {
	if search.AmountOfItemsAcrossAnswers == 0 {
		return
	}

	m.networkSearchOverlap.Observe(
		float64(search.AmountOfRepeatedItemsAcrossAnswers) /
			float64(search.AmountOfItemsAcrossAnswers),
	)
}

func (m *NetworkSearchMetrics) NetworkSearchFoundNoAskablePeers(context.Context) {
	m.peersAskedPerNetworkSearch.Observe(0)
}

func (m *NetworkSearchMetrics) NetworkSearchFoundNoIndexedTerm(context.Context) {
	m.searchesWithoutIndexedTerm.Inc()
}
