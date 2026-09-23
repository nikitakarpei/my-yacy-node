// Package wordjoined finds documents that match the whole query when no single
// peer holds every query word, by joining what the peers of each word hold.
package wordjoined

import (
	"context"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReplicaAsks interface {
	AskForSearchDocuments(
		ctx context.Context,
		asks []peerasks.SearchDocumentsAsk,
	) peerasks.SearchDocumentsAskOutcomes
}

type PeerAsks interface {
	AskForURLMetadata(
		ctx context.Context,
		asks []peerasks.URLMetadataAsk,
	) []peerasks.AnsweredURLMetadataAsk
}

type PeerJudgements interface {
	StandingsOf(
		ctx context.Context,
		peers []peerjudgements.PeerAtVersion,
	) peerjudgements.PeerStandings
	Add(ctx context.Context, judgedPeers []peerjudgements.JudgedPeer)
}

type Spread struct {
	replicaAsks                    ReplicaAsks
	peerAsks                       PeerAsks
	peerJudgements                 PeerJudgements
	urlMetadataAskDocumentsCeiling int
	documentsToMatchCeiling        int
	peerItemsCeiling               int
	partitions                     yacymodel.DHTRingPartitions
	amountOfPeersHoldingOneWord    int
	observer                       WordJoinedSpreadObserver
}

//nolint:revive // argument-limit: the spread takes its asks, judgements, ceilings, ring and observer
func New(
	replicaAsks ReplicaAsks,
	peerAsks PeerAsks,
	peerJudgements PeerJudgements,
	urlMetadataAskDocumentsCeiling int,
	documentsToMatchCeiling int,
	peerItemsCeiling int,
	partitions yacymodel.DHTRingPartitions,
	amountOfPeersHoldingOneWord int,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		replicaAsks:                    replicaAsks,
		peerAsks:                       peerAsks,
		peerJudgements:                 peerJudgements,
		urlMetadataAskDocumentsCeiling: urlMetadataAskDocumentsCeiling,
		documentsToMatchCeiling:        documentsToMatchCeiling,
		peerItemsCeiling:               peerItemsCeiling,
		partitions:                     partitions,
		amountOfPeersHoldingOneWord:    amountOfPeersHoldingOneWord,
		observer:                       observer,
	}
}

func (spread Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) queryanswers.AnsweredQuery {
	startedAt := time.Now()

	peerStandings := spread.peerJudgements.StandingsOf(
		ctx, peersAtTheirVersionFrom(chosenPeersPerQueryWord),
	)
	abstractsRound := spread.askForAbstracts(
		ctx,
		query,
		chosenPeersLeastUsefulForTheCrossCheckFirst(chosenPeersPerQueryWord, peerStandings),
	)
	crossCheckCandidates := crossCheckCandidatesIn(abstractsRound, spread.partitions)
	crossCheckRound := spread.askToCrossCheck(
		ctx,
		crossCheckCandidates,
		abstractsRound,
		peerStandings,
	)
	judgedPeers := crossCheckJudgementsIn(crossCheckRound)
	spread.peerJudgements.Add(ctx, judgedPeers)
	joinedDocuments := joinedDocumentsFrom(abstractsRound, crossCheckRound)
	answeredSearchDocumentsAsks := answeredSearchDocumentsAsksAcross(
		abstractsRound, crossCheckRound,
	)
	urlMetadataRound := spread.askForURLMetadata(
		ctx, abstractsRound, answeredSearchDocumentsAsks, joinedDocuments,
	)

	spread.observer.WordJoinedSpreadPerformed(ctx, performedWordJoinedSpreadFrom(
		abstractsRound,
		crossCheckRound,
		peerStandings,
		judgedPeers,
		joinedDocuments,
		urlMetadataRound,
		time.Since(startedAt),
	))

	return answeredQueryFrom(
		query,
		abstractsRound,
		answeredSearchDocumentsAsks,
		joinedDocuments,
		urlMetadataRound,
	)
}

func (spread Spread) askForAbstracts(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) abstractsRound {
	asks := abstractsAsksFor(query, chosenPeersPerQueryWord, spread.peerItemsCeiling)
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheAbstracts)
	defer endRound()
	askOutcomes := spread.replicaAsks.AskForSearchDocuments(roundContext, asks)
	answeredAsks := askOutcomes.AnsweredAsks()

	return abstractsRound{
		queryWords:   query.WordHashes(),
		peersAsked:   peersAskedIn(askOutcomes.AsksPut()),
		answeredAsks: answeredAsks,
		queryWordsFewestDocumentsFirst: queryWordsFewestDocumentsFirstFrom(
			query.WordHashes(), askOutcomes, spread.partitions,
		),
		compoundWords: compoundWordsAcrossReplicasFrom(
			query.CompoundWords, askOutcomes, spread.partitions,
		),
		amountOfPeersPerDocument: amountOfPeersPerDocumentOf(answeredAsks),
	}
}

const (
	roundsLeftAtTheAbstracts   = 3
	roundsLeftAtTheCrossCheck  = 2
	roundsLeftAtTheURLMetadata = 1
)

func contextOfRound(ctx context.Context, roundsLeft int) (context.Context, context.CancelFunc) {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, time.Until(deadline)/time.Duration(roundsLeft))
}

func (spread Spread) askToCrossCheck(
	ctx context.Context,
	candidates crossCheckCandidates,
	abstractsRound abstractsRound,
	peerStandings peerjudgements.PeerStandings,
) crossCheckRound {
	asks := crossCheckAsksFor(
		candidates,
		abstractsRound.peersAsked,
		peerStandings,
		spread.documentsToMatchCeiling,
		spread.peerItemsCeiling,
	)
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheCrossCheck)
	defer endRound()
	askOutcomes := spread.replicaAsks.AskForSearchDocuments(roundContext, asks)

	return crossCheckRound{
		candidates:   candidates,
		asks:         askOutcomes.AsksPut(),
		answeredAsks: askOutcomes.AnsweredAsks(),
	}
}

func answeredSearchDocumentsAsksAcross(
	abstractsRound abstractsRound,
	crossCheckRound crossCheckRound,
) []peerasks.AnsweredSearchDocumentsAsk {
	return slices.Concat(
		abstractsRound.answeredAsks, crossCheckRound.answeredAsks,
	)
}

func (spread Spread) askForURLMetadata(
	ctx context.Context,
	abstractsRound abstractsRound,
	answeredSearchDocumentsAsks []peerasks.AnsweredSearchDocumentsAsk,
	joinedDocuments distinctDocuments,
) urlMetadataRound {
	documentsWithoutMetadata := documentsWithoutMetadataAmong(
		joinedDocuments, answeredSearchDocumentsAsks,
	)
	asks := urlMetadataAsksFor(
		abstractsRound.documentsInTheMostAbstractsFirstAmong(documentsWithoutMetadata),
		abstractsRound.answeredAsks,
		spread.urlMetadataAskDocumentsCeiling,
		spread.amountOfPeersHoldingOneWord,
	)
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheURLMetadata)
	defer endRound()

	return urlMetadataRound{
		documentsWithoutMetadata: documentsWithoutMetadata,
		asks:                     asks,
		answeredAsks:             spread.peerAsks.AskForURLMetadata(roundContext, asks),
	}
}
