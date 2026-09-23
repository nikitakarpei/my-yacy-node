// Package applog reports what one word joined spread found to the service log.
package applog

import (
	"context"
	"log/slog"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const (
	msgWordJoinedSpreadPerformed = "word joined spread performed"
	msgPeerJudged                = "peer judged"
)

type WordJoinedSpreadLog struct{}

func (WordJoinedSpreadLog) WordJoinedSpreadPerformed(
	ctx context.Context,
	spread wordjoined.PerformedWordJoinedSpread,
) {
	logPerformedSpread(ctx, spread)
	logJudgedPeers(ctx, spread.CrossCheckedDocumentsRound.JudgedPeers)
}

func logPerformedSpread(ctx context.Context, spread wordjoined.PerformedWordJoinedSpread) {
	slog.LogAttrs(
		ctx,
		slog.LevelDebug,
		msgWordJoinedSpreadPerformed,
		slices.Concat(
			attributesOfAbstractsRound(spread.AbstractsRound),
			attributesOfCrossCheckedDocumentsRound(spread.CrossCheckedDocumentsRound),
			attributesOfURLMetadataRound(spread.URLMetadataRound),
			[]slog.Attr{
				slog.Any(
					"amountOfPeersPerStanding", amountOfPeersPerStandingOf(spread.PeerStandings),
				),
				slog.Duration("timeSpent", spread.TimeSpent),
			},
		)...,
	)
}

func attributesOfAbstractsRound(
	round wordjoined.PerformedAbstractsRound,
) []slog.Attr {
	return []slog.Attr{
		slog.Int("amountOfQueryWords", round.AmountOfQueryWords),
		slog.Int("amountOfCompoundWords", round.AmountOfCompoundWords),
		slog.Int("amountOfQueryWordsHeldByNoPeer", round.AmountOfQueryWordsHeldByNoPeer),
		slog.Int("amountOfQueryWordsWithCompleteAbstracts", round.AmountOfQueryWordsWithCompleteAbstracts),
		slog.Int("amountOfPeersWithANonEmptyAbstract", round.AmountOfPeersWithANonEmptyAbstract),
		slog.String("leadingQueryWordChoice", string(round.LeadingQueryWordChoice)),
		slog.Int(
			"amountOfDocumentsInTheAbstractsOfTheLeadingQueryWord",
			round.AmountOfDocumentsInTheAbstractsOfTheLeadingQueryWord,
		),
		slog.Int(
			"amountOfPartitionsWithABetterLeadingQueryWord",
			round.AmountOfPartitionsWithABetterLeadingQueryWord,
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

func attributesOfCrossCheckedDocumentsRound(
	round wordjoined.PerformedCrossCheckedDocumentsRound,
) []slog.Attr {
	return []slog.Attr{
		slog.Int(
			"amountOfDocumentsSentForCrossChecking",
			round.AmountOfDocumentsSentForCrossChecking,
		),
		slog.Int(
			"amountOfCrossCheckCandidatesNoPeerTook",
			round.AmountOfCrossCheckCandidatesNoPeerTook,
		),
		slog.Int(
			"amountOfCrossCheckCandidatesRuledOutByAFullListing",
			round.AmountOfCrossCheckCandidatesRuledOutByAFullListing,
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
		slog.Any("amountOfPeersPerJudgement", amountOfPeersPerJudgementOf(round.JudgedPeers)),
	}
}

func amountOfPeersPerJudgementOf(judgedPeers []peerjudgements.JudgedPeer) map[string]int {
	amountOfPeersPerJudgement := map[string]int{}
	for _, judgedPeer := range judgedPeers {
		amountOfPeersPerJudgement[string(judgedPeer.Judgement)]++
	}

	return amountOfPeersPerJudgement
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

func amountOfPeersPerStandingOf(peerStandings []peerjudgements.PeerStanding) map[string]int {
	amountOfPeersPerStanding := map[string]int{}
	for _, peerStanding := range peerStandings {
		amountOfPeersPerStanding[string(peerStanding.Standing)]++
	}

	return amountOfPeersPerStanding
}

func logJudgedPeers(ctx context.Context, judgedPeers []peerjudgements.JudgedPeer) {
	for _, judgedPeer := range judgedPeers {
		slog.LogAttrs(ctx, slog.LevelDebug, msgPeerJudged,
			slog.String("peer", judgedPeer.Peer.String()),
			slog.Any("versionClaimed", judgedPeer.Version),
			slog.String("judgement", string(judgedPeer.Judgement)),
			slog.String("question", string(wordjoined.ListsOnlyTheCrossCheckedDocuments)),
		)
	}
}
