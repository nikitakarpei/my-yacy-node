// Package prometheus reports how a word joined spread performed, as metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/budgetbuckets"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const (
	amountOfRatioBuckets = 11
	ratioBucketWidth     = 0.1
	labelJoin            = "join"
	joinFoundNoDocument  = "no document"
	joinFoundDocuments   = "documents"
)

type WordJoinedSpreadMetrics struct {
	joinsThatFoundDocuments         prometheusclient.Counter
	joinsThatFoundNoDocument        prometheusclient.Counter
	queryWords                      queryWordMetrics
	matchingWordAsks                matchingWordAskMetrics
	urlMetadataAsks                 urlMetadataAskMetrics
	wordJoinedSpreadDurationSeconds prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *WordJoinedSpreadMetrics {
	wordJoinedSpreads := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spreads_total",
		Help: "Word joined spreads, by whether the join found a document.",
	}, []string{labelJoin})
	metrics := &WordJoinedSpreadMetrics{
		joinsThatFoundDocuments:  wordJoinedSpreads.WithLabelValues(joinFoundDocuments),
		joinsThatFoundNoDocument: wordJoinedSpreads.WithLabelValues(joinFoundNoDocument),
		queryWords:               queryWordMetricsRegisteredIn(registry),
		matchingWordAsks:         matchingWordAskMetricsRegisteredIn(registry),
		urlMetadataAsks:          urlMetadataAskMetricsRegisteredIn(registry),
		wordJoinedSpreadDurationSeconds: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name:    "yacydhtsearch_word_joined_spread_duration_seconds",
				Help:    "Word joined spread duration in seconds.",
				Buckets: budgetbuckets.DurationBucketsFor(queryBudget),
			},
		),
	}
	registry.MustRegister(wordJoinedSpreads, metrics.wordJoinedSpreadDurationSeconds)

	return metrics
}

func ratioHistogramNamed(name string, help string) prometheusclient.Histogram {
	return prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: prometheusclient.LinearBuckets(0, ratioBucketWidth, amountOfRatioBuckets),
	})
}

func (m *WordJoinedSpreadMetrics) WordJoinedSpreadPerformed(
	_ context.Context,
	spread wordjoined.PerformedWordJoinedSpread,
) {
	m.matchingWordAsks.observeMatchingWordAsks(spread.MatchingWordAskKindPerPartition)
	m.queryWords.observeQueryWords(spread)
	m.urlMetadataAsks.observeURLMetadataAsks(spread)
	m.countJoin(spread.AmountOfJoinedDocuments)
	m.observeWordJoinedSpreadDuration(spread.TimeSpent)
}

func (m *WordJoinedSpreadMetrics) countJoin(amountOfJoinedDocuments int) {
	if amountOfJoinedDocuments == 0 {
		m.joinsThatFoundNoDocument.Inc()

		return
	}
	m.joinsThatFoundDocuments.Inc()
}

func (m *WordJoinedSpreadMetrics) observeWordJoinedSpreadDuration(timeSpent time.Duration) {
	m.wordJoinedSpreadDurationSeconds.Observe(timeSpent.Seconds())
}
