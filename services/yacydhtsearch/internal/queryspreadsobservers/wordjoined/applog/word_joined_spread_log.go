// Package applog reports what one word joined spread found to the service log.
package applog

import (
	"context"
	"log/slog"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/matchingwords"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordasks"
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
			attributesOfWordAsks(spread.WordAsks),
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
		slog.Any(
			"amountOfPartitionsPerOtherWordAsks",
			amountOfPartitionsPerKindIn(spread.MatchingWordAskKindPerPartition),
		),
	}
}

func amountOfPartitionsPerKindIn(kindPerPartition matchingwords.KindPerPartition) map[string]int {
	amountOfPartitionsPerKind := map[string]int{}
	for _, kind := range kindPerPartition {
		amountOfPartitionsPerKind[string(kind)]++
	}

	return amountOfPartitionsPerKind
}

func attributesOfWordAsks(wordAsks wordasks.Performed) []slog.Attr {
	return []slog.Attr{
		slog.Int(
			"amountOfPeersWithANonEmptyAbstract",
			wordAsks.AmountOfPeersWithANonEmptyAbstract,
		),
		slog.Int(
			"amountOfMatchedDocumentsAcrossAnswers",
			wordAsks.AmountOfMatchedDocumentsAcrossAnswers,
		),
		slog.Int(
			"amountOfMatchedDocumentsWithAPosting",
			wordAsks.AmountOfMatchedDocumentsWithAPosting,
		),
		slog.Any("amountOfDocumentsHeldInEachAnswer", wordAsks.AmountOfDocumentsHeldInEachAnswer),
	}
}

func attributesOfURLMetadataAsks(urlMetadataAsks urlmetadataasks.Performed) []slog.Attr {
	return []slog.Attr{
		slog.Int("amountOfLookedUpDocuments", urlMetadataAsks.AmountOfAskedDocuments),
		slog.Int(
			"amountOfLookedUpDocumentsWithMetadata",
			urlMetadataAsks.AmountOfAskedDocumentsWithMetadata,
		),
		slog.String("urlMetadataLookupEndReason", string(urlMetadataAsks.EndReason)),
		slog.Int("amountOfDocumentsCutOffDuringLookup", urlMetadataAsks.AmountOfDocumentsCutOff),
	}
}
