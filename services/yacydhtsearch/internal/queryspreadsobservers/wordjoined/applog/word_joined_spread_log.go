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
	matchedAndHeldDocumentsRound := spread.MatchedAndHeldDocumentsRound
	heldDocumentsRound := spread.HeldDocumentsRound
	urlMetadataRound := spread.URLMetadataRound
	slog.DebugContext(ctx, msgWordJoinedSpreadPerformed,
		slog.Int("amountOfQueryWords", matchedAndHeldDocumentsRound.AmountOfQueryWords),
		slog.Int(
			"amountOfQueryWordsHeldByNoPeer",
			matchedAndHeldDocumentsRound.AmountOfQueryWordsHeldByNoPeer,
		),
		slog.Int("amountOfCutOffQueryWords", matchedAndHeldDocumentsRound.AmountOfCutOffQueryWords),
		slog.Int("amountOfPeersAsked", matchedAndHeldDocumentsRound.AmountOfPeersAsked),
		slog.Int(
			"amountOfPeersThatAnswered",
			matchedAndHeldDocumentsRound.AmountOfPeersThatAnswered,
		),
		slog.Int(
			"amountOfPeersHoldingAQueryWord",
			matchedAndHeldDocumentsRound.AmountOfPeersHoldingAQueryWord,
		),
		slog.Int("amountOfAnchorDocuments", matchedAndHeldDocumentsRound.AmountOfAnchorDocuments),
		slog.Int(
			"amountOfDocumentsPastTheHeldDocumentsCeiling",
			heldDocumentsRound.AmountOfDocumentsPastTheHeldDocumentsCeiling,
		),
		slog.Int(
			"amountOfPeersAskedForHeldDocuments",
			heldDocumentsRound.AmountOfPeersAskedForHeldDocuments,
		),
		slog.Int(
			"amountOfPeersThatAnsweredHeldDocuments",
			heldDocumentsRound.AmountOfPeersThatAnsweredHeldDocuments,
		),
		slog.Int(
			"amountOfEmptyHeldDocumentsAnswers",
			heldDocumentsRound.AmountOfEmptyHeldDocumentsAnswers,
		),
		slog.Int(
			"amountOfJoinedDocumentsBeforeTheHeldDocumentsAsks",
			heldDocumentsRound.AmountOfJoinedDocumentsBeforeTheHeldDocumentsAsks,
		),
		slog.Int("amountOfJoinedDocuments", heldDocumentsRound.AmountOfJoinedDocuments),
		slog.Int(
			"amountOfJoinedDocumentsWithMetadata",
			urlMetadataRound.AmountOfJoinedDocumentsWithMetadata,
		),
		slog.Int(
			"amountOfMatchedDocumentsAcrossAnswers",
			matchedAndHeldDocumentsRound.AmountOfMatchedDocumentsAcrossAnswers,
		),
		slog.Int(
			"amountOfMatchedDocumentsCountedByAPeer",
			matchedAndHeldDocumentsRound.AmountOfMatchedDocumentsCountedByAPeer,
		),
		slog.Any(
			"amountOfDocumentsHeldInEachAnswer",
			matchedAndHeldDocumentsRound.AmountOfDocumentsHeldInEachAnswer,
		),
		slog.Int(
			"amountOfDocumentsAskedMetadataFor",
			urlMetadataRound.AmountOfDocumentsAskedMetadataFor,
		),
		slog.Int(
			"amountOfAskedDocumentsWithMetadata",
			urlMetadataRound.AmountOfAskedDocumentsWithMetadata,
		),
		slog.Duration("timeSpent", spread.TimeSpent),
	)
}
