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
			attributesOfMatchedAndHeldDocumentsRound(spread.MatchedAndHeldDocumentsRound),
			attributesOfCrossCheckedDocumentsRound(spread.CrossCheckedDocumentsRound),
			attributesOfURLMetadataRound(spread.URLMetadataRound),
			[]slog.Attr{slog.Duration("timeSpent", spread.TimeSpent)},
		)...,
	)
}

func attributesOfMatchedAndHeldDocumentsRound(
	round wordjoined.PerformedMatchedAndHeldDocumentsRound,
) []slog.Attr {
	return []slog.Attr{
		slog.Int("amountOfQueryWords", round.AmountOfQueryWords),
		slog.Int("amountOfQueryWordsHeldByNoPeer", round.AmountOfQueryWordsHeldByNoPeer),
		slog.Int("amountOfFullyListedQueryWords", round.AmountOfFullyListedQueryWords),
		slog.Int(
			"amountOfPeersAskedForMatchedAndHeldDocuments",
			round.AmountOfPeersAskedForMatchedAndHeldDocuments,
		),
		slog.Int(
			"amountOfPeersThatAnsweredMatchedAndHeldDocuments",
			round.AmountOfPeersThatAnsweredMatchedAndHeldDocuments,
		),
		slog.Int("amountOfPeersThatListedADocument", round.AmountOfPeersThatListedADocument),
		slog.String("leadingQueryWordStanding", string(round.LeadingQueryWordStanding)),
		slog.Int(
			"amountOfDocumentsListedByThePeersOfTheLeadingQueryWord",
			round.AmountOfDocumentsListedByThePeersOfTheLeadingQueryWord,
		),
		slog.Int(
			"amountOfMatchedDocumentsAcrossAnswers",
			round.AmountOfMatchedDocumentsAcrossAnswers,
		),
		slog.Int(
			"amountOfMatchedDocumentsCountedByAPeer",
			round.AmountOfMatchedDocumentsCountedByAPeer,
		),
		slog.Any("amountOfDocumentsHeldInEachAnswer", round.AmountOfDocumentsHeldInEachAnswer),
	}
}

func attributesOfCrossCheckedDocumentsRound(
	round wordjoined.PerformedCrossCheckedDocumentsRound,
) []slog.Attr {
	return []slog.Attr{
		slog.Int(
			"amountOfDocumentsPastTheCrossCheckedDocumentsCeiling",
			round.AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling,
		),
		slog.Int(
			"amountOfPeersAskedForCrossCheckedDocuments",
			round.AmountOfPeersAskedForCrossCheckedDocuments,
		),
		slog.Int(
			"amountOfPeersThatAnsweredCrossCheckedDocuments",
			round.AmountOfPeersThatAnsweredCrossCheckedDocuments,
		),
		slog.Int(
			"amountOfEmptyCrossCheckedDocumentsAnswers",
			round.AmountOfEmptyCrossCheckedDocumentsAnswers,
		),
		slog.Int("amountOfJoinedDocuments", round.AmountOfJoinedDocuments),
		slog.Int(
			"amountOfJoinedDocumentsFoundOnlyByCrossChecking",
			round.AmountOfJoinedDocumentsFoundOnlyByCrossChecking,
		),
	}
}

func attributesOfURLMetadataRound(round wordjoined.PerformedURLMetadataRound) []slog.Attr {
	return []slog.Attr{
		slog.Int(
			"amountOfJoinedDocumentsWithMetadata",
			round.AmountOfJoinedDocumentsWithMetadata,
		),
		slog.Int("amountOfLookedUpDocuments", round.AmountOfLookedUpDocuments),
		slog.Int(
			"amountOfLookedUpDocumentsWithMetadata",
			round.AmountOfLookedUpDocumentsWithMetadata,
		),
	}
}
