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
			"amountOfReplicaAsksPerPutAs",
			amountOfReplicaAsksPerPutAs(replicaAsks.WordPartitions),
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

func amountOfReplicaAsksPerPutAs(
	wordPartitions []replicaasks.PerformedWordPartition,
) map[replicaasks.PutAs]int {
	amountOfReplicaAsks := make(map[replicaasks.PutAs]int, len(wordPartitions))
	for _, wordPartition := range wordPartitions {
		for _, replicaAsk := range wordPartition.Asks {
			amountOfReplicaAsks[replicaAsk.PutAs]++
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
