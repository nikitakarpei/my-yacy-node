package prometheus

import (
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const (
	amountOfRoundDurationBuckets = 35
	labelRound                   = "round"
	roundDiscovery               = "discovery"
	roundCrossCheck              = "cross-check"
	roundURLMetadata             = "URL metadata"
)

type reportedRoundTime interface {
	Get() (wordjoined.RoundTime, bool)
}

type roundTimeMetrics struct {
	discoveryRound   timeOfOneRoundMetrics
	crossCheckRound  timeOfOneRoundMetrics
	urlMetadataRound timeOfOneRoundMetrics
}

func roundTimeMetricsRegisteredIn(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) roundTimeMetrics {
	roundDurationSeconds := prometheusclient.NewHistogramVec(prometheusclient.HistogramOpts{
		Name: "yacydhtsearch_word_joined_spread_round_duration_seconds",
		Help: "Time one round of a word joined spread took, in seconds, by the round. " +
			"A round that asks no peer is not counted.",
		Buckets: durationBucketsWithin(queryBudget, amountOfRoundDurationBuckets),
	}, []string{labelRound})
	roundBudgetSeconds := prometheusclient.NewHistogramVec(prometheusclient.HistogramOpts{
		Name: "yacydhtsearch_word_joined_spread_round_budget_seconds",
		Help: "Time one round of a word joined spread was given, in seconds, by the round. " +
			"A round that asks no peer is not counted.",
		Buckets: durationBucketsWithin(queryBudget, amountOfRoundDurationBuckets),
	}, []string{labelRound})
	registry.MustRegister(roundDurationSeconds, roundBudgetSeconds)

	return roundTimeMetrics{
		discoveryRound: timeOfOneRoundMetrics{
			durationSeconds: roundDurationSeconds.WithLabelValues(roundDiscovery),
			budgetSeconds:   roundBudgetSeconds.WithLabelValues(roundDiscovery),
		},
		crossCheckRound: timeOfOneRoundMetrics{
			durationSeconds: roundDurationSeconds.WithLabelValues(roundCrossCheck),
			budgetSeconds:   roundBudgetSeconds.WithLabelValues(roundCrossCheck),
		},
		urlMetadataRound: timeOfOneRoundMetrics{
			durationSeconds: roundDurationSeconds.WithLabelValues(roundURLMetadata),
			budgetSeconds:   roundBudgetSeconds.WithLabelValues(roundURLMetadata),
		},
	}
}

func (m roundTimeMetrics) observeRoundTimes(spread wordjoined.PerformedWordJoinedSpread) {
	m.discoveryRound.observeRoundTime(spread.DiscoveryRound.Time)
	m.crossCheckRound.observeRoundTime(spread.CrossCheckRound.Time)
	m.urlMetadataRound.observeRoundTime(spread.URLMetadataRound.Time)
}

type timeOfOneRoundMetrics struct {
	durationSeconds prometheusclient.Observer
	budgetSeconds   prometheusclient.Observer
}

func (m timeOfOneRoundMetrics) observeRoundTime(roundTime reportedRoundTime) {
	timeOfTheRound, asked := roundTime.Get()
	if !asked {
		return
	}
	m.durationSeconds.Observe(timeOfTheRound.TimeSpent.Seconds())
	budget, bounded := timeOfTheRound.Budget.Get()
	if !bounded {
		return
	}
	m.budgetSeconds.Observe(budget.Seconds())
}
