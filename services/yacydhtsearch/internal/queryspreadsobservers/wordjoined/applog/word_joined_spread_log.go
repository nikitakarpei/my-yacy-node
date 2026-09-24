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
			attributesOfDiscoveryRound(spread.DiscoveryRound),
			[]slog.Attr{slog.Int("amountOfJoinedDocuments", spread.AmountOfJoinedDocuments)},
			attributesOfURLMetadataLookupRound(spread.URLMetadataLookupRound),
			[]slog.Attr{slog.Duration("timeSpent", spread.TimeSpent)},
		)...,
	)
}

func attributesOfDiscoveryRound(
	round wordjoined.PerformedDiscoveryRound,
) []slog.Attr {
	return []slog.Attr{
		slog.Int("amountOfQueryWords", round.AmountOfQueryWords),
		slog.Int("amountOfCompoundWords", round.AmountOfCompoundWords),
		slog.Int("amountOfQueryWordsHeldByNoPeer", round.AmountOfQueryWordsHeldByNoPeer),
		slog.Uint64("sampledPartition", uint64(round.SampledPartition)),
		slog.Int("amountOfQueryWordsWithASample", round.AmountOfQueryWordsWithASample),
		slog.Int("amountOfPeersWithANonEmptyAbstract", round.AmountOfPeersWithANonEmptyAbstract),
		slog.String("leadingQueryWordChoice", string(round.LeadingQueryWordChoice)),
		slog.Int(
			"amountOfDocumentsOfTheLeadingQueryWord",
			round.AmountOfDocumentsOfTheLeadingQueryWord,
		),
		slog.Any(
			"amountOfPartitionsPerOtherWordAsks",
			amountOfPartitionsPerOtherWordAsksFrom(round.OtherWordAsksPerPartition),
		),
		slog.Int(
			"amountOfMatchedDocumentsAcrossAnswers",
			round.AmountOfMatchedDocumentsAcrossAnswers,
		),
		slog.Int(
			"amountOfMatchedDocumentsWithAPosting",
			round.AmountOfMatchedDocumentsWithAPosting,
		),
		slog.Any("amountOfDocumentsHeldInEachAnswer", round.AmountOfDocumentsHeldInEachAnswer),
	}
}

func amountOfPartitionsPerOtherWordAsksFrom(
	otherWordAsksPerPartition map[uint]wordjoined.OtherWordAsks,
) map[string]int {
	amountOfPartitionsPerOtherWordAsks := map[string]int{}
	for _, otherWordAsks := range otherWordAsksPerPartition {
		amountOfPartitionsPerOtherWordAsks[string(otherWordAsks)]++
	}

	return amountOfPartitionsPerOtherWordAsks
}

func attributesOfURLMetadataLookupRound(
	round wordjoined.PerformedURLMetadataLookupRound,
) []slog.Attr {
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
		slog.String("urlMetadataLookupEnd", string(round.End)),
	}
}
