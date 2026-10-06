// Package applog reports what one word joined spread found to the service log.
package applog

import (
	"context"
	"log/slog"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
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
			[]slog.Attr{
				slog.Int("amountOfJoinedDocuments", spread.AmountOfJoinedDocuments),
				slog.Int(
					"amountOfJoinedDocumentsWithMetadata",
					spread.AmountOfJoinedDocumentsWithMetadata,
				),
				slog.Duration("timeSpent", spread.TimeSpent),
			},
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
