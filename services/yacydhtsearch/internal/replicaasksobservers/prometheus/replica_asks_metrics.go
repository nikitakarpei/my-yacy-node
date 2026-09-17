// Package prometheus reports as metrics how long the replica asks of one spread
// took, what settled each word partition, what put each ask to a replica, and
// how many documents a settled word partition listed.
package prometheus

import (
	"context"
	"math"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
)

const (
	labelEndedBy                   = "ended_by"
	labelSettledBy                 = "settled_by"
	labelPutAs                     = "put_as"
	durationBucketRatio            = 1.6
	bucketsUpToBudget              = 15
	documentsListedBucketCeiling   = 1024.0
	amountOfDocumentsListedBuckets = 11
)

var overBudgetShares = []float64{1.25, 1.5, 2}

type ReplicaAsksMetrics struct {
	replicaAsksDurationSecondsPerEndedBy map[replicaasks.EndedBy]prometheusclient.Observer
	wordPartitionsPerSettledBy           map[replicaasks.SettledBy]prometheusclient.Counter
	replicaAsksPerPutAs                  map[replicaasks.PutAs]prometheusclient.Counter
	wordPartitionDocumentsListed         prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *ReplicaAsksMetrics {
	replicaAsksDurationSeconds := prometheusclient.NewHistogramVec(
		prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_replica_asks_duration_seconds",
			Help:    "Time the replica asks of one spread call took, by what ended them.",
			Buckets: replicaAsksDurationBucketsFor(queryBudget),
		},
		[]string{labelEndedBy},
	)
	wordPartitions := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_partitions_total",
		Help: "Settled word partitions, by what settled the word partition.",
	}, []string{labelSettledBy})
	replicaAsks := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_replica_asks_total",
		Help: "Asks put to a replica of a word partition, by what put the ask.",
	}, []string{labelPutAs})
	wordPartitionDocumentsListed := prometheusclient.NewHistogram(
		prometheusclient.HistogramOpts{
			Name: "yacydhtsearch_word_partition_documents_listed",
			Help: "Documents the replicas of one settled word partition listed for the word.",
			Buckets: bucketsFromNoneTo(
				documentsListedBucketCeiling, amountOfDocumentsListedBuckets,
			),
		},
	)
	registry.MustRegister(
		replicaAsksDurationSeconds, wordPartitions, replicaAsks, wordPartitionDocumentsListed,
	)

	return &ReplicaAsksMetrics{
		replicaAsksDurationSecondsPerEndedBy: replicaAsksDurationSecondsPerEndedByFrom(
			replicaAsksDurationSeconds,
		),
		wordPartitionsPerSettledBy:   wordPartitionsPerSettledByFrom(wordPartitions),
		replicaAsksPerPutAs:          replicaAsksPerPutAsFrom(replicaAsks),
		wordPartitionDocumentsListed: wordPartitionDocumentsListed,
	}
}

func replicaAsksDurationBucketsFor(queryBudget time.Duration) []float64 {
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

func bucketsFromNoneTo(ceiling float64, amountOfBuckets int) []float64 {
	return append(
		[]float64{0},
		prometheusclient.ExponentialBucketsRange(1, ceiling, amountOfBuckets)...,
	)
}

func replicaAsksDurationSecondsPerEndedByFrom(
	replicaAsksDurationSeconds *prometheusclient.HistogramVec,
) map[replicaasks.EndedBy]prometheusclient.Observer {
	//exhaustive:enforce
	return map[replicaasks.EndedBy]prometheusclient.Observer{
		replicaasks.EndedByCoverage: replicaAsksDurationSeconds.WithLabelValues(
			string(replicaasks.EndedByCoverage),
		),
		replicaasks.EndedByDeadline: replicaAsksDurationSeconds.WithLabelValues(
			string(replicaasks.EndedByDeadline),
		),
	}
}

func wordPartitionsPerSettledByFrom(
	wordPartitions *prometheusclient.CounterVec,
) map[replicaasks.SettledBy]prometheusclient.Counter {
	//exhaustive:enforce
	return map[replicaasks.SettledBy]prometheusclient.Counter{
		replicaasks.SettledByFirst: wordPartitions.WithLabelValues(
			string(replicaasks.SettledByFirst),
		),
		replicaasks.SettledByHedge: wordPartitions.WithLabelValues(
			string(replicaasks.SettledByHedge),
		),
		replicaasks.SettledByAfterAnEmptyAnswer: wordPartitions.WithLabelValues(
			string(replicaasks.SettledByAfterAnEmptyAnswer),
		),
		replicaasks.SettledByAfterAFailure: wordPartitions.WithLabelValues(
			string(replicaasks.SettledByAfterAFailure),
		),
		replicaasks.SettledByExhausted: wordPartitions.WithLabelValues(
			string(replicaasks.SettledByExhausted),
		),
		replicaasks.SettledByDeadline: wordPartitions.WithLabelValues(
			string(replicaasks.SettledByDeadline),
		),
	}
}

func replicaAsksPerPutAsFrom(
	replicaAsks *prometheusclient.CounterVec,
) map[replicaasks.PutAs]prometheusclient.Counter {
	//exhaustive:enforce
	return map[replicaasks.PutAs]prometheusclient.Counter{
		replicaasks.PutAsFirst: replicaAsks.WithLabelValues(string(replicaasks.PutAsFirst)),
		replicaasks.PutAsHedge: replicaAsks.WithLabelValues(string(replicaasks.PutAsHedge)),
		replicaasks.PutAsAfterAnEmptyAnswer: replicaAsks.WithLabelValues(
			string(replicaasks.PutAsAfterAnEmptyAnswer),
		),
		replicaasks.PutAsAfterAFailure: replicaAsks.WithLabelValues(
			string(replicaasks.PutAsAfterAFailure),
		),
	}
}

func (m *ReplicaAsksMetrics) ReplicaAsksPerformed(
	_ context.Context,
	replicaAsks replicaasks.PerformedReplicaAsks,
) {
	m.replicaAsksDurationSecondsPerEndedBy[replicaAsks.EndedBy].Observe(
		replicaAsks.TimeSpent.Seconds(),
	)
	for _, wordPartition := range replicaAsks.WordPartitions {
		m.countWordPartition(wordPartition)
	}
}

func (m *ReplicaAsksMetrics) countWordPartition(
	wordPartition replicaasks.PerformedWordPartition,
) {
	m.wordPartitionsPerSettledBy[wordPartition.SettledBy].Inc()
	m.wordPartitionDocumentsListed.Observe(float64(wordPartition.AmountOfDocumentsListed))
	for _, replicaAsk := range wordPartition.Asks {
		m.replicaAsksPerPutAs[replicaAsk.PutAs].Inc()
	}
}
