// Package applog reports what one word joined search found to the service log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const msgWordJoinedSearchPerformed = "word joined search performed"

type WordJoinedSearchLog struct{}

func (WordJoinedSearchLog) WordJoinedSearchPerformed(
	ctx context.Context,
	search wordjoined.PerformedWordJoinedSearch,
) {
	slog.DebugContext(ctx, msgWordJoinedSearchPerformed,
		slog.Int("amountOfQueryWords", search.AmountOfQueryWords),
		slog.Int("amountOfQueryWordsNoPeerHeld", search.AmountOfQueryWordsNoPeerHeld),
		slog.Int(
			"amountOfPeersAskedForHeldDocuments",
			search.AmountOfPeersAskedForHeldDocuments,
		),
		slog.Int(
			"amountOfPeersThatNamedHeldDocuments",
			search.AmountOfPeersThatNamedHeldDocuments,
		),
		slog.Int("amountOfPeersHoldingAQueryWord", search.AmountOfPeersHoldingAQueryWord),
		slog.Int("amountOfJoinedDocuments", search.AmountOfJoinedDocuments),
		slog.Int(
			"amountOfJoinedDocumentsAlreadyReported",
			search.AmountOfJoinedDocumentsAlreadyReported,
		),
		slog.Int("amountOfReportedItems", search.AmountOfReportedItems),
		slog.Int(
			"amountOfReportedItemsWithAPosting",
			search.AmountOfReportedItemsWithAPosting,
		),
		slog.Any(
			"documentsEachPeerHoldsForAQueryWord",
			search.DocumentsEachPeerHoldsForAQueryWord,
		),
		slog.Int(
			"amountOfDocumentsToAskMetadataFor",
			search.AmountOfDocumentsToAskMetadataFor,
		),
		slog.Int("amountOfDocumentsThatCameBack", search.AmountOfDocumentsThatCameBack),
		slog.Duration("timeSpent", search.TimeSpent),
	)
}
