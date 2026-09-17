// Package prometheus reports how a word joined spread performed, as metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const (
	amountOfDurationBuckets                  = 12
	shortestDurationBucketShareOfQueryBudget = 1.0 / 1024
	longestDurationBucketShareOfQueryBudget  = 2.0
	amountOfRatioBuckets                     = 11
	ratioBucketWidth                         = 0.1
	labelJoin                                = "join"
	joinFoundNoDocument                      = "no document"
	joinFoundDocuments                       = "documents"
	labelLeadingQueryWordStanding            = "leading_query_word_standing"
)

type WordJoinedSpreadMetrics struct {
	joinsPerLeadingQueryWordStanding map[wordjoined.LeadingQueryWordStanding]leadingQueryWordStandingJoins
	matchedAndHeldDocumentsRound     matchedAndHeldDocumentsRoundMetrics
	crossCheckedDocumentsRound       crossCheckedDocumentsRoundMetrics
	urlMetadataRound                 urlMetadataRoundMetrics
	wordJoinedSpreadDurationSeconds  prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *WordJoinedSpreadMetrics {
	wordJoinedSpreads := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spreads_total",
		Help: "Word joined spreads, by whether the join found a document and by which " +
			"query word led the join.",
	}, []string{labelJoin, labelLeadingQueryWordStanding})
	//exhaustive:enforce
	joinsPerLeadingQueryWordStanding := map[wordjoined.LeadingQueryWordStanding]leadingQueryWordStandingJoins{
		wordjoined.RarestFullyListedQueryWord: leadingQueryWordStandingJoinsFrom(
			wordJoinedSpreads, wordjoined.RarestFullyListedQueryWord,
		),
		wordjoined.MoreCommonFullyListedQueryWord: leadingQueryWordStandingJoinsFrom(
			wordJoinedSpreads, wordjoined.MoreCommonFullyListedQueryWord,
		),
		wordjoined.RarestPartlyListedQueryWord: leadingQueryWordStandingJoinsFrom(
			wordJoinedSpreads, wordjoined.RarestPartlyListedQueryWord,
		),
	}
	metrics := &WordJoinedSpreadMetrics{
		joinsPerLeadingQueryWordStanding: joinsPerLeadingQueryWordStanding,
		matchedAndHeldDocumentsRound:     matchedAndHeldDocumentsRoundMetricsRegisteredIn(registry),
		crossCheckedDocumentsRound:       crossCheckedDocumentsRoundMetricsRegisteredIn(registry),
		urlMetadataRound:                 urlMetadataRoundMetricsRegisteredIn(registry),
		wordJoinedSpreadDurationSeconds: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_word_joined_spread_duration_seconds",
				Help: "Word joined spread duration in seconds.",
				Buckets: prometheusclient.ExponentialBucketsRange(
					queryBudget.Seconds()*shortestDurationBucketShareOfQueryBudget,
					queryBudget.Seconds()*longestDurationBucketShareOfQueryBudget,
					amountOfDurationBuckets,
				),
			},
		),
	}
	registry.MustRegister(wordJoinedSpreads, metrics.wordJoinedSpreadDurationSeconds)

	return metrics
}

type leadingQueryWordStandingJoins struct {
	joinsThatFoundDocuments  prometheusclient.Counter
	joinsThatFoundNoDocument prometheusclient.Counter
}

func leadingQueryWordStandingJoinsFrom(
	wordJoinedSpreads *prometheusclient.CounterVec,
	leadingQueryWordStanding wordjoined.LeadingQueryWordStanding,
) leadingQueryWordStandingJoins {
	return leadingQueryWordStandingJoins{
		joinsThatFoundDocuments: wordJoinedSpreads.WithLabelValues(
			joinFoundDocuments, string(leadingQueryWordStanding),
		),
		joinsThatFoundNoDocument: wordJoinedSpreads.WithLabelValues(
			joinFoundNoDocument, string(leadingQueryWordStanding),
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
	m.matchedAndHeldDocumentsRound.observeMatchedAndHeldDocumentsRound(
		spread.MatchedAndHeldDocumentsRound,
	)
	m.crossCheckedDocumentsRound.observeCrossCheckedDocumentsRound(
		spread.CrossCheckedDocumentsRound,
		spread.MatchedAndHeldDocumentsRound,
	)
	m.urlMetadataRound.observeURLMetadataRound(
		spread.URLMetadataRound,
		spread.CrossCheckedDocumentsRound,
	)
	m.countJoin(
		spread.MatchedAndHeldDocumentsRound.LeadingQueryWordStanding,
		spread.CrossCheckedDocumentsRound,
	)
	m.observeWordJoinedSpreadDuration(spread.TimeSpent)
}

func (m *WordJoinedSpreadMetrics) countJoin(
	leadingQueryWordStanding wordjoined.LeadingQueryWordStanding,
	crossCheckedDocumentsRound wordjoined.PerformedCrossCheckedDocumentsRound,
) {
	joinsOfTheLeadingQueryWordStanding := m.joinsPerLeadingQueryWordStanding[leadingQueryWordStanding]
	if crossCheckedDocumentsRound.AmountOfJoinedDocuments == 0 {
		joinsOfTheLeadingQueryWordStanding.joinsThatFoundNoDocument.Inc()

		return
	}
	joinsOfTheLeadingQueryWordStanding.joinsThatFoundDocuments.Inc()
}

func (m *WordJoinedSpreadMetrics) observeWordJoinedSpreadDuration(timeSpent time.Duration) {
	m.wordJoinedSpreadDurationSeconds.Observe(timeSpent.Seconds())
}
