// Package applog reports to the service log what the replica asks of one spread
// put to the replicas of each word partition.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
)

const msgReplicaAsksPerformed = "replica asks performed"

type ReplicaAsksLog struct{}

func (ReplicaAsksLog) ReplicaAsksPerformed(
	ctx context.Context,
	replicaAsks replicaasks.PerformedReplicaAsks,
) {
	slog.LogAttrs(
		ctx,
		slog.LevelDebug,
		msgReplicaAsksPerformed,
		slog.String("endedBy", string(replicaAsks.EndedBy)),
		slog.Duration("timeSpent", replicaAsks.TimeSpent),
		slog.Any(
			"amountOfWordPartitionsPerSettledBy",
			amountOfWordPartitionsPerSettledBy(replicaAsks.WordPartitions),
		),
		slog.Any(
			"amountOfWordPartitionsCoveredPerAskPutOn",
			amountOfWordPartitionsCoveredPerAskPutOn(replicaAsks.WordPartitions),
		),
		slog.Any(
			"amountOfReplicaAsksPerPutOn",
			amountOfReplicaAsksPerPutOn(replicaAsks.WordPartitions),
		),
		slog.Int(
			"amountOfDocumentsListedAcrossWordPartitions",
			amountOfDocumentsListedAcrossWordPartitions(replicaAsks.WordPartitions),
		),
	)
}

func amountOfWordPartitionsPerSettledBy(
	wordPartitions []replicaasks.PerformedWordPartition,
) map[replicaasks.SettledBy]int {
	amountOfWordPartitions := make(map[replicaasks.SettledBy]int, len(wordPartitions))
	for _, wordPartition := range wordPartitions {
		amountOfWordPartitions[wordPartition.SettledBy]++
	}

	return amountOfWordPartitions
}

func amountOfWordPartitionsCoveredPerAskPutOn(
	wordPartitions []replicaasks.PerformedWordPartition,
) map[replicaasks.PutOn]int {
	amountOfWordPartitions := make(map[replicaasks.PutOn]int, len(wordPartitions))
	for _, wordPartition := range wordPartitions {
		if wordPartition.SettledBy == replicaasks.SettledByCoverage {
			amountOfWordPartitions[wordPartition.CoveringAskPutOn]++
		}
	}

	return amountOfWordPartitions
}

func amountOfReplicaAsksPerPutOn(
	wordPartitions []replicaasks.PerformedWordPartition,
) map[replicaasks.PutOn]int {
	amountOfReplicaAsks := make(map[replicaasks.PutOn]int, len(wordPartitions))
	for _, wordPartition := range wordPartitions {
		for _, putOn := range wordPartition.AsksPutOn {
			amountOfReplicaAsks[putOn]++
		}
	}

	return amountOfReplicaAsks
}

func amountOfDocumentsListedAcrossWordPartitions(
	wordPartitions []replicaasks.PerformedWordPartition,
) int {
	amountOfDocuments := 0
	for _, wordPartition := range wordPartitions {
		amountOfDocuments += wordPartition.AmountOfDocumentsListed
	}

	return amountOfDocuments
}
