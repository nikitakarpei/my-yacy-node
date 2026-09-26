// Package prometheus reports as metrics how long the replica asks of one spread
// took, what settled each word partition and what put the ask that covered it,
// what put each ask to a replica, and how many documents a settled word
// partition listed.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/budgetbuckets"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

const (
	labelEndedBy                   = "ended_by"
	labelSettledBy                 = "settled_by"
	labelCoveringAskPutOn          = "covering_ask_put_on"
	labelPutOn                     = "put_on"
	documentsListedBucketCeiling   = 1024.0
	amountOfDocumentsListedBuckets = 11
)

type ReplicaAsksMetrics struct {
	replicaAsksDurationSecondsPerEndedBy map[wordpartitionasks.EndedBy]prometheusclient.Observer
	wordPartitionsPerSettledBy           map[wordpartitionasks.SettledBy]map[wordpartitionasks.PutOn]prometheusclient.Counter
	replicaAsksPerPutOn                  map[wordpartitionasks.PutOn]prometheusclient.Counter
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
			Buckets: budgetbuckets.DurationBucketsFor(queryBudget),
		},
		[]string{labelEndedBy},
	)
	wordPartitions := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_partitions_total",
		Help: "Settled word partitions, by what settled the word partition " +
			"and, when covered, by what put the ask that covered it.",
	}, []string{labelSettledBy, labelCoveringAskPutOn})
	replicaAsks := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_replica_asks_total",
		Help: "Asks put to a replica of a word partition, by what put the ask.",
	}, []string{labelPutOn})
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
		replicaAsksPerPutOn:          replicaAsksPerPutOnFrom(replicaAsks),
		wordPartitionDocumentsListed: wordPartitionDocumentsListed,
	}
}

func bucketsFromNoneTo(ceiling float64, amountOfBuckets int) []float64 {
	return append(
		[]float64{0},
		prometheusclient.ExponentialBucketsRange(1, ceiling, amountOfBuckets)...,
	)
}

func replicaAsksDurationSecondsPerEndedByFrom(
	replicaAsksDurationSeconds *prometheusclient.HistogramVec,
) map[wordpartitionasks.EndedBy]prometheusclient.Observer {
	//exhaustive:enforce
	return map[wordpartitionasks.EndedBy]prometheusclient.Observer{
		wordpartitionasks.EndedByCoverage: replicaAsksDurationSeconds.WithLabelValues(
			string(wordpartitionasks.EndedByCoverage),
		),
		wordpartitionasks.EndedByDeadline: replicaAsksDurationSeconds.WithLabelValues(
			string(wordpartitionasks.EndedByDeadline),
		),
	}
}

func wordPartitionsPerSettledByFrom(
	wordPartitions *prometheusclient.CounterVec,
) map[wordpartitionasks.SettledBy]map[wordpartitionasks.PutOn]prometheusclient.Counter {
	//exhaustive:enforce
	return map[wordpartitionasks.SettledBy]map[wordpartitionasks.PutOn]prometheusclient.Counter{
		wordpartitionasks.SettledByCoverage: wordPartitionsCoveredPerAskPutOnFrom(wordPartitions),
		wordpartitionasks.SettledByNoReplicaLeft: {
			"": wordPartitions.WithLabelValues(
				string(wordpartitionasks.SettledByNoReplicaLeft),
				"",
			),
		},
		wordpartitionasks.SettledByDeadline: {
			"": wordPartitions.WithLabelValues(string(wordpartitionasks.SettledByDeadline), ""),
		},
	}
}

func wordPartitionsCoveredPerAskPutOnFrom(
	wordPartitions *prometheusclient.CounterVec,
) map[wordpartitionasks.PutOn]prometheusclient.Counter {
	covered := func(putOn wordpartitionasks.PutOn) prometheusclient.Counter {
		return wordPartitions.WithLabelValues(
			string(wordpartitionasks.SettledByCoverage),
			string(putOn),
		)
	}
	//exhaustive:enforce
	return map[wordpartitionasks.PutOn]prometheusclient.Counter{
		wordpartitionasks.PutOnStart:       covered(wordpartitionasks.PutOnStart),
		wordpartitionasks.PutOnHedgeDelay:  covered(wordpartitionasks.PutOnHedgeDelay),
		wordpartitionasks.PutOnEmptyAnswer: covered(wordpartitionasks.PutOnEmptyAnswer),
		wordpartitionasks.PutOnFailure:     covered(wordpartitionasks.PutOnFailure),
	}
}

func replicaAsksPerPutOnFrom(
	replicaAsks *prometheusclient.CounterVec,
) map[wordpartitionasks.PutOn]prometheusclient.Counter {
	//exhaustive:enforce
	return map[wordpartitionasks.PutOn]prometheusclient.Counter{
		wordpartitionasks.PutOnStart: replicaAsks.WithLabelValues(
			string(wordpartitionasks.PutOnStart),
		),
		wordpartitionasks.PutOnHedgeDelay: replicaAsks.WithLabelValues(
			string(wordpartitionasks.PutOnHedgeDelay),
		),
		wordpartitionasks.PutOnEmptyAnswer: replicaAsks.WithLabelValues(
			string(wordpartitionasks.PutOnEmptyAnswer),
		),
		wordpartitionasks.PutOnFailure: replicaAsks.WithLabelValues(
			string(wordpartitionasks.PutOnFailure),
		),
	}
}

func (m *ReplicaAsksMetrics) ReplicaAsksPerformed(
	_ context.Context,
	replicaAsks wordpartitionasks.PerformedReplicaAsks,
) {
	m.replicaAsksDurationSecondsPerEndedBy[replicaAsks.EndedBy].Observe(
		replicaAsks.TimeSpent.Seconds(),
	)
	for _, wordPartition := range replicaAsks.WordPartitions {
		m.countWordPartition(wordPartition)
	}
}

func (m *ReplicaAsksMetrics) countWordPartition(
	wordPartition wordpartitionasks.PerformedWordPartition,
) {
	m.wordPartitionsPerSettledBy[wordPartition.SettledBy][wordPartition.CoveringAskPutOn].Inc()
	m.wordPartitionDocumentsListed.Observe(float64(wordPartition.AmountOfDocumentsListed))
	for _, putOn := range wordPartition.AsksPutOn {
		m.replicaAsksPerPutOn[putOn].Inc()
	}
}
