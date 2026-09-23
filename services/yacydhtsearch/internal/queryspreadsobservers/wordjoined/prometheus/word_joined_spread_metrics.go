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
	labelLeadingQueryWordChoice              = "leading_query_word_choice"
)

type WordJoinedSpreadMetrics struct {
	joinsPerLeadingQueryWordChoice  map[wordjoined.LeadingQueryWordChoice]leadingQueryWordChoiceJoins
	abstractsRound                  abstractsRoundMetrics
	crossCheckRound                 crossCheckRoundMetrics
	peerJudgements                  peerJudgementsMetrics
	urlMetadataRound                urlMetadataRoundMetrics
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
		wordjoined.RarestQueryWordWithCompleteAbstracts: leadingQueryWordChoiceJoinsFrom(
			wordJoinedSpreads, wordjoined.RarestQueryWordWithCompleteAbstracts,
		),
		wordjoined.MoreCommonQueryWordWithCompleteAbstracts: leadingQueryWordChoiceJoinsFrom(
			wordJoinedSpreads, wordjoined.MoreCommonQueryWordWithCompleteAbstracts,
		),
		wordjoined.RarestQueryWordWithoutCompleteAbstracts: leadingQueryWordChoiceJoinsFrom(
			wordJoinedSpreads, wordjoined.RarestQueryWordWithoutCompleteAbstracts,
		),
	}
	metrics := &WordJoinedSpreadMetrics{
		joinsPerLeadingQueryWordChoice: joinsPerLeadingQueryWordChoice,
		abstractsRound:                 abstractsRoundMetricsRegisteredIn(registry),
		crossCheckRound:                crossCheckRoundMetricsRegisteredIn(registry),
		peerJudgements:                 peerJudgementsMetricsRegisteredIn(registry),
		urlMetadataRound:               urlMetadataRoundMetricsRegisteredIn(registry),
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
	m.abstractsRound.observeAbstractsRound(
		spread.AbstractsRound,
	)
	m.crossCheckRound.observeCrossCheckRound(
		spread.CrossCheckRound,
	)
	m.peerJudgements.countStandingsAndJudgements(spread)
	m.urlMetadataRound.observeURLMetadataRound(
		spread.URLMetadataRound,
		spread.CrossCheckRound,
	)
	m.countJoin(
		spread.AbstractsRound.LeadingQueryWordChoice,
		spread.CrossCheckRound,
	)
	m.observeWordJoinedSpreadDuration(spread.TimeSpent)
}

func (m *WordJoinedSpreadMetrics) countJoin(
	leadingQueryWordChoice wordjoined.LeadingQueryWordChoice,
	crossCheckRound wordjoined.PerformedCrossCheckRound,
) {
	joinsOfTheLeadingQueryWordChoice := m.joinsPerLeadingQueryWordChoice[leadingQueryWordChoice]
	if crossCheckRound.AmountOfJoinedDocuments == 0 {
		joinsOfTheLeadingQueryWordChoice.joinsThatFoundNoDocument.Inc()

		return
	}
	joinsOfTheLeadingQueryWordChoice.joinsThatFoundDocuments.Inc()
}

func (m *WordJoinedSpreadMetrics) observeWordJoinedSpreadDuration(timeSpent time.Duration) {
	m.wordJoinedSpreadDurationSeconds.Observe(timeSpent.Seconds())
}
