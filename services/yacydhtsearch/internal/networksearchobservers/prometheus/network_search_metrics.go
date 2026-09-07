// Package prometheus reports how many peers a network search reached, and its
// completeness, overlap and duration, as metrics.
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
	peersAskedPerNetworkSearch   prometheusclient.Histogram
	networkSearchCompleteness    prometheusclient.Histogram
	networkSearchOverlap         prometheusclient.Histogram
	networkSearchDurationSeconds prometheusclient.Histogram
}

func New(registry prometheusclient.Registerer, queryBudget time.Duration) *NetworkSearchMetrics {
	metrics := &NetworkSearchMetrics{
		networkSearchesPerformed: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_network_searches_performed_total",
			Help: "Network searches performed. One that found no askable peer is not.",
		}),
		peersAskedPerNetworkSearch: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_network_search_peers_asked",
			Help:    "Peers asked for one network search. Zero means it found no askable peer.",
			Buckets: prometheusclient.ExponentialBucketsRange(1, 128, 8),
		}),
		networkSearchCompleteness: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_network_search_completeness_ratio",
			Help:    "Share of asked peers that returned at least one item.",
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
		metrics.peersAskedPerNetworkSearch,
		metrics.networkSearchCompleteness,
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
	m.networkSearchCompleteness.Observe(
		float64(search.AmountOfAnsweringPeers) / float64(search.AmountOfAskedPeers),
	)
	m.networkSearchOverlap.Observe(overlapOf(search))
}

func overlapOf(search networksearch.PerformedNetworkSearch) float64 {
	if search.AmountOfItemsAcrossAnswers == 0 {
		return 0
	}

	return float64(
		search.AmountOfRepeatedItemsAcrossAnswers,
	) / float64(
		search.AmountOfItemsAcrossAnswers,
	)
}

func (m *NetworkSearchMetrics) NetworkSearchFoundNoAskablePeers(context.Context) {
	m.peersAskedPerNetworkSearch.Observe(0)
}
