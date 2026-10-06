// Package applog reports to the service log how the partitions of a word joined
// spread were asked for the other words, and what the document asks of one query
// got back.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
)

const (
	msgPartitionsAskedForTheOtherWords = "partitions asked for the other words"
	msgDocumentAsksPerformed           = "document asks performed"
)

type DocumentAsksLog struct{}

func (DocumentAsksLog) AskedAmongTheDocuments(
	ctx context.Context,
	decisionPerPartition documentasks.DocumentsToMatchDecisionPerPartition,
) {
	slog.DebugContext(
		ctx,
		msgPartitionsAskedForTheOtherWords,
		slog.Any(
			"amountOfPartitionsPerOtherWordAsks",
			decisionPerPartition.AmountOfPartitionsPerDecision(),
		),
	)
}

func (DocumentAsksLog) DocumentAsksPerformed(
	ctx context.Context,
	performed documentasks.Performed,
) {
	slog.DebugContext(
		ctx,
		msgDocumentAsksPerformed,
		slog.Int(
			"amountOfPeersWithANonEmptyAbstract",
			performed.AmountOfPeersWithANonEmptyAbstract,
		),
		slog.Int(
			"amountOfListedDocumentsWithMetadata",
			performed.AmountOfListedDocumentsWithMetadata,
		),
		slog.Int(
			"amountOfListedDocumentsWithAPosting",
			performed.AmountOfListedDocumentsWithAPosting,
		),
		slog.Any(
			"amountOfDocumentsHeldInEachAnswer",
			performed.AmountOfDocumentsHeldInEachAnswer,
		),
	)
}
