// Package prometheus reports as metrics, by what the asks asked for, how long
// the replica asks of one spread took, what settled each word partition and
// what put the ask that covered it, what put each ask to a replica, and how
// many documents a settled word partition listed.
package prometheus

import (
	"context"
	"math"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
)

const (
	labelAskedFor                  = "asked_for"
	labelEndedBy                   = "ended_by"
	labelSettledBy                 = "settled_by"
	labelCoveringAskPutOn          = "covering_ask_put_on"
	labelPutOn                     = "put_on"
	durationBucketRatio            = 1.6
	bucketsUpToBudget              = 15
	documentsListedBucketCeiling   = 1024.0
	amountOfDocumentsListedBuckets = 11
)

var overBudgetShares = []float64{1.25, 1.5, 2}

type ReplicaAsksMetrics struct {
	metricsPerAskedFor map[peerasks.AskedFor]replicaAsksMetricsOfAskedFor
}

type replicaAsksMetricsOfAskedFor struct {
	replicaAsksDurationSecondsPerEndedBy map[replicaasks.EndedBy]prometheusclient.Observer
	wordPartitionsPerSettledBy           map[replicaasks.SettledBy]map[replicaasks.PutOn]prometheusclient.Counter
	replicaAsksPerPutOn                  map[replicaasks.PutOn]prometheusclient.Counter
	wordPartitionDocumentsListed         prometheusclient.Observer
}

type replicaAsksVectors struct {
	replicaAsksDurationSeconds   *prometheusclient.HistogramVec
	wordPartitions               *prometheusclient.CounterVec
	replicaAsks                  *prometheusclient.CounterVec
	wordPartitionDocumentsListed *prometheusclient.HistogramVec
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *ReplicaAsksMetrics {
	vectors := replicaAsksVectorsRegisteredIn(registry, queryBudget)

	//exhaustive:enforce
	return &ReplicaAsksMetrics{
		metricsPerAskedFor: map[peerasks.AskedFor]replicaAsksMetricsOfAskedFor{
			peerasks.MatchedDocuments: vectors.metricsOf(peerasks.MatchedDocuments),
			peerasks.MatchedAndHeldDocuments: vectors.metricsOf(
				peerasks.MatchedAndHeldDocuments,
			),
			peerasks.CrossCheckedDocuments: vectors.metricsOf(peerasks.CrossCheckedDocuments),
			peerasks.URLMetadata:           vectors.metricsOf(peerasks.URLMetadata),
		},
	}
}

func replicaAsksVectorsRegisteredIn(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) replicaAsksVectors {
	vectors := replicaAsksVectors{
		replicaAsksDurationSeconds: prometheusclient.NewHistogramVec(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_replica_asks_duration_seconds",
				Help: "Time the replica asks of one spread call took, by what they asked for " +
					"and what ended them.",
				Buckets: replicaAsksDurationBucketsFor(queryBudget),
			},
			[]string{labelAskedFor, labelEndedBy},
		),
		wordPartitions: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_word_partitions_total",
			Help: "Settled word partitions, by what the asks asked for, by what settled the " +
				"word partition and, when covered, by what put the ask that covered it.",
		}, []string{labelAskedFor, labelSettledBy, labelCoveringAskPutOn}),
		replicaAsks: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_replica_asks_total",
			Help: "Asks put to a replica of a word partition, by what the ask asked for and " +
				"what put the ask.",
		}, []string{labelAskedFor, labelPutOn}),
		wordPartitionDocumentsListed: prometheusclient.NewHistogramVec(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_word_partition_documents_listed",
				Help: "Documents the replicas of one settled word partition listed for the " +
					"word, by what the asks asked for.",
				Buckets: bucketsFromNoneTo(
					documentsListedBucketCeiling, amountOfDocumentsListedBuckets,
				),
			},
			[]string{labelAskedFor},
		),
	}
	registry.MustRegister(
		vectors.replicaAsksDurationSeconds,
		vectors.wordPartitions,
		vectors.replicaAsks,
		vectors.wordPartitionDocumentsListed,
	)

	return vectors
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

func (vectors replicaAsksVectors) metricsOf(
	askedFor peerasks.AskedFor,
) replicaAsksMetricsOfAskedFor {
	askedForLabel := prometheusclient.Labels{labelAskedFor: string(askedFor)}

	return replicaAsksMetricsOfAskedFor{
		replicaAsksDurationSecondsPerEndedBy: replicaAsksDurationSecondsPerEndedByFrom(
			vectors.replicaAsksDurationSeconds.MustCurryWith(askedForLabel),
		),
		wordPartitionsPerSettledBy: wordPartitionsPerSettledByFrom(
			vectors.wordPartitions.MustCurryWith(askedForLabel),
		),
		replicaAsksPerPutOn: replicaAsksPerPutOnFrom(
			vectors.replicaAsks.MustCurryWith(askedForLabel),
		),
		wordPartitionDocumentsListed: vectors.wordPartitionDocumentsListed.WithLabelValues(
			string(askedFor),
		),
	}
}

func replicaAsksDurationSecondsPerEndedByFrom(
	replicaAsksDurationSeconds prometheusclient.ObserverVec,
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
) map[replicaasks.SettledBy]map[replicaasks.PutOn]prometheusclient.Counter {
	//exhaustive:enforce
	return map[replicaasks.SettledBy]map[replicaasks.PutOn]prometheusclient.Counter{
		replicaasks.SettledByCoverage: wordPartitionsCoveredPerAskPutOnFrom(wordPartitions),
		replicaasks.SettledByNoReplicaLeft: {
			"": wordPartitions.WithLabelValues(string(replicaasks.SettledByNoReplicaLeft), ""),
		},
		replicaasks.SettledByDeadline: {
			"": wordPartitions.WithLabelValues(string(replicaasks.SettledByDeadline), ""),
		},
	}
}

func wordPartitionsCoveredPerAskPutOnFrom(
	wordPartitions *prometheusclient.CounterVec,
) map[replicaasks.PutOn]prometheusclient.Counter {
	covered := func(putOn replicaasks.PutOn) prometheusclient.Counter {
		return wordPartitions.WithLabelValues(string(replicaasks.SettledByCoverage), string(putOn))
	}
	//exhaustive:enforce
	return map[replicaasks.PutOn]prometheusclient.Counter{
		replicaasks.PutOnStart:             covered(replicaasks.PutOnStart),
		replicaasks.PutOnHedgeDelay:        covered(replicaasks.PutOnHedgeDelay),
		replicaasks.PutOnNonCoveringAnswer: covered(replicaasks.PutOnNonCoveringAnswer),
		replicaasks.PutOnFailure:           covered(replicaasks.PutOnFailure),
	}
}

func replicaAsksPerPutOnFrom(
	replicaAsks *prometheusclient.CounterVec,
) map[replicaasks.PutOn]prometheusclient.Counter {
	//exhaustive:enforce
	return map[replicaasks.PutOn]prometheusclient.Counter{
		replicaasks.PutOnStart: replicaAsks.WithLabelValues(string(replicaasks.PutOnStart)),
		replicaasks.PutOnHedgeDelay: replicaAsks.WithLabelValues(
			string(replicaasks.PutOnHedgeDelay),
		),
		replicaasks.PutOnNonCoveringAnswer: replicaAsks.WithLabelValues(
			string(replicaasks.PutOnNonCoveringAnswer),
		),
		replicaasks.PutOnFailure: replicaAsks.WithLabelValues(string(replicaasks.PutOnFailure)),
	}
}

func (m *ReplicaAsksMetrics) ReplicaAsksPerformed(
	_ context.Context,
	replicaAsks replicaasks.PerformedReplicaAsks,
) {
	metrics := m.metricsPerAskedFor[replicaAsks.AskedFor]
	metrics.replicaAsksDurationSecondsPerEndedBy[replicaAsks.EndedBy].Observe(
		replicaAsks.TimeSpent.Seconds(),
	)
	for _, wordPartition := range replicaAsks.WordPartitions {
		metrics.countWordPartition(wordPartition)
	}
}

func (m replicaAsksMetricsOfAskedFor) countWordPartition(
	wordPartition replicaasks.SettledWordPartition,
) {
	m.wordPartitionsPerSettledBy[wordPartition.SettledBy][wordPartition.CoveringAskPutOn].Inc()
	m.wordPartitionDocumentsListed.Observe(float64(wordPartition.AmountOfDocumentsListed))
	for _, putOn := range wordPartition.AsksPutOn {
		m.replicaAsksPerPutOn[putOn].Inc()
	}
}
