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
	amountOfRatioBuckets        = 11
	ratioBucketWidth            = 0.1
	labelJoin                   = "join"
	joinFoundNoDocument         = "no document"
	joinFoundDocuments          = "documents"
	labelLeadingQueryWordChoice = "leading_query_word_choice"
)

type WordJoinedSpreadMetrics struct {
	joinsPerLeadingQueryWordChoice  map[wordjoined.LeadingQueryWordChoice]leadingQueryWordChoiceJoins
	discoveryRound                  discoveryRoundMetrics
	urlMetadataLookupRound          urlMetadataLookupRoundMetrics
	wordJoinedSpreadDurationSeconds prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *WordJoinedSpreadMetrics {
	wordJoinedSpreads := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spreads_total",
		Help: "Word joined spreads, by whether the join found a document and by which " +
			"query word led the join.",
	}, []string{labelJoin, labelLeadingQueryWordChoice})
	//exhaustive:enforce
	joinsPerLeadingQueryWordChoice := map[wordjoined.LeadingQueryWordChoice]leadingQueryWordChoiceJoins{
		wordjoined.RarestQueryWordWithASample: leadingQueryWordChoiceJoinsFrom(
			wordJoinedSpreads, wordjoined.RarestQueryWordWithASample,
		),
		wordjoined.RarestQueryWordWithoutASample: leadingQueryWordChoiceJoinsFrom(
			wordJoinedSpreads, wordjoined.RarestQueryWordWithoutASample,
		),
		wordjoined.RarestQueryWordRemembered: leadingQueryWordChoiceJoinsFrom(
			wordJoinedSpreads, wordjoined.RarestQueryWordRemembered,
		),
	}
	metrics := &WordJoinedSpreadMetrics{
		joinsPerLeadingQueryWordChoice: joinsPerLeadingQueryWordChoice,
		discoveryRound:                 discoveryRoundMetricsRegisteredIn(registry),
		urlMetadataLookupRound:         urlMetadataLookupRoundMetricsRegisteredIn(registry),
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

type leadingQueryWordChoiceJoins struct {
	joinsThatFoundDocuments  prometheusclient.Counter
	joinsThatFoundNoDocument prometheusclient.Counter
}

func leadingQueryWordChoiceJoinsFrom(
	wordJoinedSpreads *prometheusclient.CounterVec,
	leadingQueryWordChoice wordjoined.LeadingQueryWordChoice,
) leadingQueryWordChoiceJoins {
	return leadingQueryWordChoiceJoins{
		joinsThatFoundDocuments: wordJoinedSpreads.WithLabelValues(
			joinFoundDocuments, string(leadingQueryWordChoice),
		),
		joinsThatFoundNoDocument: wordJoinedSpreads.WithLabelValues(
			joinFoundNoDocument, string(leadingQueryWordChoice),
		),
	}
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
	m.discoveryRound.observeDiscoveryRound(
		spread.DiscoveryRound,
	)
	m.urlMetadataLookupRound.observeURLMetadataLookupRound(
		spread.URLMetadataLookupRound,
		spread.AmountOfJoinedDocuments,
	)
	m.countJoin(
		spread.DiscoveryRound.LeadingQueryWordChoice,
		spread.AmountOfJoinedDocuments,
	)
	m.observeWordJoinedSpreadDuration(spread.TimeSpent)
}

func (m *WordJoinedSpreadMetrics) countJoin(
	leadingQueryWordChoice wordjoined.LeadingQueryWordChoice,
	amountOfJoinedDocuments int,
) {
	joinsOfTheLeadingQueryWordChoice := m.joinsPerLeadingQueryWordChoice[leadingQueryWordChoice]
	if amountOfJoinedDocuments == 0 {
		joinsOfTheLeadingQueryWordChoice.joinsThatFoundNoDocument.Inc()

		return
	}
	joinsOfTheLeadingQueryWordChoice.joinsThatFoundDocuments.Inc()
}

func (m *WordJoinedSpreadMetrics) observeWordJoinedSpreadDuration(timeSpent time.Duration) {
	m.wordJoinedSpreadDurationSeconds.Observe(timeSpent.Seconds())
}
