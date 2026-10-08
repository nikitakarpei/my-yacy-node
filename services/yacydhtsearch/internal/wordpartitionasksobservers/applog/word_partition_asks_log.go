// Package applog reports to the service log what the word partition asks of one
// query put to the replicas of each word partition.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

const msgWordPartitionAsksPerformed = "word partition asks performed"

type WordPartitionAsksLog struct{}

func (WordPartitionAsksLog) WordPartitionAsksPerformed(
	ctx context.Context,
	wordPartitionAsks wordpartitionasks.PerformedWordPartitionAsks,
) {
	slog.LogAttrs(
		ctx,
		slog.LevelDebug,
		msgWordPartitionAsksPerformed,
		slog.String("endedBy", string(wordPartitionAsks.EndedBy)),
		slog.Duration("timeSpent", wordPartitionAsks.TimeSpent),
		slog.Any(
			"amountOfWordPartitionsPerSettledBy",
			amountOfWordPartitionsPerSettledBy(wordPartitionAsks.WordPartitions),
		),
		slog.Any(
			"amountOfWordPartitionsCoveredPerAskPutOn",
			amountOfWordPartitionsCoveredPerAskPutOn(wordPartitionAsks.WordPartitions),
		),
		slog.Any(
			"amountOfReplicaAsksPerPutOn",
			amountOfReplicaAsksPerPutOn(wordPartitionAsks.WordPartitions),
		),
		slog.Int(
			"amountOfDocumentsListedAcrossWordPartitions",
			amountOfDocumentsListedAcrossWordPartitions(wordPartitionAsks.WordPartitions),
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
