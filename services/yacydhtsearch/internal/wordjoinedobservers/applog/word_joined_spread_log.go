// Package applog reports what one word joined spread found to the service log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const msgWordJoinedSpreadPerformed = "word joined spread performed"

type WordJoinedSpreadLog struct{}

func (WordJoinedSpreadLog) WordJoinedSpreadPerformed(
	ctx context.Context,
	spread wordjoined.PerformedWordJoinedSpread,
) {
	slog.DebugContext(ctx, msgWordJoinedSpreadPerformed,
		slog.Int("amountOfQueryWords", spread.AmountOfQueryWords),
		slog.Int("amountOfQueryWordsHeldByNoPeer", spread.AmountOfQueryWordsHeldByNoPeer),
		slog.Int("amountOfPeersAsked", spread.AmountOfPeersAsked),
		slog.Int("amountOfPeersThatAnswered", spread.AmountOfPeersThatAnswered),
		slog.Int("amountOfPeersHoldingAQueryWord", spread.AmountOfPeersHoldingAQueryWord),
		slog.Int("amountOfJoinedDocuments", spread.AmountOfJoinedDocuments),
		slog.Int(
			"amountOfJoinedDocumentsWithMetadata",
			spread.AmountOfJoinedDocumentsWithMetadata,
		),
		slog.Int(
			"amountOfMatchedDocumentsAcrossAnswers",
			spread.AmountOfMatchedDocumentsAcrossAnswers,
		),
		slog.Int(
			"amountOfMatchedDocumentsCountedByAPeer",
			spread.AmountOfMatchedDocumentsCountedByAPeer,
		),
		slog.Any(
			"amountOfDocumentsHeldInEachAnswer",
			spread.AmountOfDocumentsHeldInEachAnswer,
		),
		slog.Int(
			"amountOfDocumentsAskedMetadataFor",
			spread.AmountOfDocumentsAskedMetadataFor,
		),
		slog.Int(
			"amountOfAskedDocumentsWithMetadata",
			spread.AmountOfAskedDocumentsWithMetadata,
		),
		slog.Duration("timeSpent", spread.TimeSpent),
	)
}
