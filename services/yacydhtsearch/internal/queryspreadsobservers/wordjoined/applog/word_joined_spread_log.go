// Package applog reports what one word joined spread found to the service log.
package applog

import (
	"context"
	"log/slog"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
)

const msgWordJoinedSpreadPerformed = "word joined spread performed"

type WordJoinedSpreadLog struct{}

func (WordJoinedSpreadLog) WordJoinedSpreadPerformed(
	ctx context.Context,
	spread wordjoined.PerformedWordJoinedSpread,
) {
	slog.LogAttrs(
		ctx,
		slog.LevelDebug,
		msgWordJoinedSpreadPerformed,
		slices.Concat(
			attributesOfQueryWords(spread),
			attributesOfDocumentAsks(spread.DocumentAsks),
			[]slog.Attr{
				slog.Int("amountOfJoinedDocuments", spread.AmountOfJoinedDocuments),
				slog.Int(
					"amountOfJoinedDocumentsWithMetadata",
					spread.AmountOfJoinedDocumentsWithMetadata,
				),
			},
			attributesOfURLMetadataAsks(spread.URLMetadataAsks),
			[]slog.Attr{slog.Duration("timeSpent", spread.TimeSpent)},
		)...,
	)
}

func attributesOfQueryWords(spread wordjoined.PerformedWordJoinedSpread) []slog.Attr {
	return []slog.Attr{
		slog.Int("amountOfQueryWords", spread.AmountOfQueryWords),
		slog.Int("amountOfCompoundWords", spread.AmountOfCompoundWords),
		slog.Int("amountOfQueryWordsHeldByNoPeer", spread.AmountOfQueryWordsHeldByNoPeer),
		slog.Int(
			"amountOfDocumentsOfTheLeadingQueryWord",
			spread.AmountOfDocumentsOfTheLeadingWord,
		),
	}
}

func attributesOfDocumentAsks(documentAsks documentasks.Performed) []slog.Attr {
	return []slog.Attr{
		slog.Int(
			"amountOfPeersWithANonEmptyAbstract",
			documentAsks.AmountOfPeersWithANonEmptyAbstract,
		),
		slog.Int(
			"amountOfListedDocumentsWithMetadata",
			documentAsks.AmountOfListedDocumentsWithMetadata,
		),
		slog.Int(
			"amountOfListedDocumentsWithAPosting",
			documentAsks.AmountOfListedDocumentsWithAPosting,
		),
		slog.Any(
			"amountOfDocumentsHeldInEachAnswer",
			documentAsks.AmountOfDocumentsHeldInEachAnswer,
		),
	}
}

func attributesOfURLMetadataAsks(urlMetadataAsks urlmetadataasks.Performed) []slog.Attr {
	attributes := []slog.Attr{
		slog.Int("amountOfLookedUpDocuments", urlMetadataAsks.AmountOfLookedUpDocuments),
		slog.Int(
			"amountOfLookedUpDocumentsWithMetadata",
			urlMetadataAsks.AmountOfLookedUpDocumentsWithMetadata,
		),
		slog.String("urlMetadataLookupEndReason", string(urlMetadataAsks.EndReason)),
		slog.Int("amountOfDocumentsCutOffDuringLookup", urlMetadataAsks.AmountOfDocumentsCutOff),
	}
	if timeToFirstAsk, asked := urlMetadataAsks.TimeToFirstAsk.Get(); asked {
		attributes = append(
			attributes, slog.Duration("timeToFirstUrlMetadataAsk", timeToFirstAsk),
		)
	}

	return attributes
}
