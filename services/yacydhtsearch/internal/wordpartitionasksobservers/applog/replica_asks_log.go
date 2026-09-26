// Package applog reports to the service log what the replica asks of one spread
// put to the replicas of each word partition.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

const msgReplicaAsksPerformed = "replica asks performed"

type ReplicaAsksLog struct{}

func (ReplicaAsksLog) ReplicaAsksPerformed(
	ctx context.Context,
	replicaAsks wordpartitionasks.PerformedReplicaAsks,
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
	wordPartitions []wordpartitionasks.PerformedWordPartition,
) map[wordpartitionasks.SettledBy]int {
	amountOfWordPartitions := make(map[wordpartitionasks.SettledBy]int, len(wordPartitions))
	for _, wordPartition := range wordPartitions {
		amountOfWordPartitions[wordPartition.SettledBy]++
	}

	return amountOfWordPartitions
}

func amountOfWordPartitionsCoveredPerAskPutOn(
	wordPartitions []wordpartitionasks.PerformedWordPartition,
) map[wordpartitionasks.PutOn]int {
	amountOfWordPartitions := make(map[wordpartitionasks.PutOn]int, len(wordPartitions))
	for _, wordPartition := range wordPartitions {
		if wordPartition.SettledBy == wordpartitionasks.SettledByCoverage {
			amountOfWordPartitions[wordPartition.CoveringAskPutOn]++
		}
	}

	return amountOfWordPartitions
}

func amountOfReplicaAsksPerPutOn(
	wordPartitions []wordpartitionasks.PerformedWordPartition,
) map[wordpartitionasks.PutOn]int {
	amountOfReplicaAsks := make(map[wordpartitionasks.PutOn]int, len(wordPartitions))
	for _, wordPartition := range wordPartitions {
		for _, putOn := range wordPartition.AsksPutOn {
			amountOfReplicaAsks[putOn]++
		}
	}

	return amountOfReplicaAsks
}

func amountOfDocumentsListedAcrossWordPartitions(
	wordPartitions []wordpartitionasks.PerformedWordPartition,
) int {
	amountOfDocuments := 0
	for _, wordPartition := range wordPartitions {
		amountOfDocuments += wordPartition.AmountOfDocumentsListed
	}

	return amountOfDocuments
}
