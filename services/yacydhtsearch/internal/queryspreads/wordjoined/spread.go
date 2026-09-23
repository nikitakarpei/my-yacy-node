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
	AskForURLMetadata(
		ctx context.Context,
		asks []peerasks.URLMetadataAsk,
	) peerasks.URLMetadataAskOutcomes
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
	peerJudgements                 PeerJudgements
	urlMetadataAskDocumentsCeiling int
	documentsToMatchCeiling        int
	peerItemsCeiling               int
	partitions                     yacymodel.DHTRingPartitions
	observer                       WordJoinedSpreadObserver
}

//nolint:revive // argument-limit: the spread takes its asks, judgements, ceilings, ring and observer
func New(
	replicaAsks ReplicaAsks,
	peerJudgements PeerJudgements,
	urlMetadataAskDocumentsCeiling int,
	documentsToMatchCeiling int,
	peerItemsCeiling int,
	partitions yacymodel.DHTRingPartitions,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		replicaAsks:                    replicaAsks,
		peerJudgements:                 peerJudgements,
		urlMetadataAskDocumentsCeiling: urlMetadataAskDocumentsCeiling,
		documentsToMatchCeiling:        documentsToMatchCeiling,
		peerItemsCeiling:               peerItemsCeiling,
		partitions:                     partitions,
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
	discoveryRound := spread.askToDiscover(
		ctx,
		query,
		chosenPeersLeastUsefulForTheCrossCheckFirst(chosenPeersPerQueryWord, peerStandings),
	)
	crossCheckCandidates := crossCheckCandidatesIn(discoveryRound, spread.partitions)
	crossCheckRound := spread.askToCrossCheck(
		ctx,
		crossCheckCandidates,
		discoveryRound,
		peerStandings,
	)
	judgedPeers := crossCheckJudgementsIn(crossCheckRound)
	spread.peerJudgements.Add(ctx, judgedPeers)
	joinedDocuments := joinedDocumentsFrom(discoveryRound, crossCheckRound)
	answeredSearchDocumentsAsks := answeredSearchDocumentsAsksAcross(
		discoveryRound, crossCheckRound,
	)
	urlMetadataRound := spread.askForURLMetadata(
		ctx, discoveryRound, answeredSearchDocumentsAsks, joinedDocuments,
	)

	spread.observer.WordJoinedSpreadPerformed(ctx, performedWordJoinedSpreadFrom(
		discoveryRound,
		crossCheckRound,
		peerStandings,
		judgedPeers,
		joinedDocuments,
		urlMetadataRound,
		time.Since(startedAt),
	))

	return answeredQueryFrom(
		query,
		discoveryRound,
		answeredSearchDocumentsAsks,
		joinedDocuments,
		urlMetadataRound,
	)
}

func (spread Spread) askToDiscover(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) discoveryRound {
	asks := discoveryAsksFor(query, chosenPeersPerQueryWord, spread.peerItemsCeiling)
	clock := roundClockStartedWithin(ctx, roundsLeftAtTheDiscovery)
	roundContext, endRound := clock.contextWithinTheBudget(ctx)
	defer endRound()
	askOutcomes := spread.replicaAsks.AskForSearchDocuments(roundContext, asks)
	answeredAsks := askOutcomes.AnsweredAsks()

	return discoveryRound{
		queryWords:   query.WordHashes(),
		peersAsked:   peersAskedIn(askOutcomes.AsksPut()),
		answeredAsks: answeredAsks,
		queryWordsFewestDocumentsFirst: queryWordsFewestDocumentsFirstFrom(
			query.WordHashes(), askOutcomes, spread.partitions,
		),
		compoundWords: compoundWordsAcrossReplicasFrom(
			query.CompoundWords, askOutcomes, spread.partitions,
		),
		holdersPerDocument: holdersPerDocumentOf(answeredAsks),
		time:               clock.roundTimeOf(len(asks)),
	}
}

func (spread Spread) askToCrossCheck(
	ctx context.Context,
	candidates crossCheckCandidates,
	discoveryRound discoveryRound,
	peerStandings peerjudgements.PeerStandings,
) crossCheckRound {
	asks := crossCheckAsksFor(
		candidates,
		discoveryRound.peersAsked,
		peerStandings,
		spread.documentsToMatchCeiling,
		spread.peerItemsCeiling,
	)
	clock := roundClockStartedWithin(ctx, roundsLeftAtTheCrossCheck)
	roundContext, endRound := clock.contextWithinTheBudget(ctx)
	defer endRound()
	askOutcomes := spread.replicaAsks.AskForSearchDocuments(roundContext, asks)

	return crossCheckRound{
		candidates:   candidates,
		asks:         askOutcomes.AsksPut(),
		answeredAsks: askOutcomes.AnsweredAsks(),
		time:         clock.roundTimeOf(len(asks)),
	}
}

func answeredSearchDocumentsAsksAcross(
	discoveryRound discoveryRound,
	crossCheckRound crossCheckRound,
) []peerasks.AnsweredSearchDocumentsAsk {
	return slices.Concat(
		discoveryRound.answeredAsks, crossCheckRound.answeredAsks,
	)
}

func (spread Spread) askForURLMetadata(
	ctx context.Context,
	discoveryRound discoveryRound,
	answeredSearchDocumentsAsks []peerasks.AnsweredSearchDocumentsAsk,
	joinedDocuments distinctDocuments,
) urlMetadataRound {
	documentsWithoutMetadata := documentsWithoutMetadataAmong(
		joinedDocuments, answeredSearchDocumentsAsks,
	)
	asks := urlMetadataAsksFor(
		discoveryRound.holdersPerDocument.mostHeldFirst(documentsWithoutMetadata),
		answeredSearchDocumentsAsks,
		spread.partitions,
		spread.urlMetadataAskDocumentsCeiling,
	)
	clock := roundClockStartedWithin(ctx, roundsLeftAtTheURLMetadata)
	roundContext, endRound := clock.contextWithinTheBudget(ctx)
	defer endRound()
	askOutcomes := spread.replicaAsks.AskForURLMetadata(roundContext, asks)

	return urlMetadataRound{
		documentsWithoutMetadata: documentsWithoutMetadata,
		asks:                     askOutcomes.AsksPut(),
		answeredAsks:             askOutcomes.AnsweredAsks(),
		time:                     clock.roundTimeOf(len(asks)),
	}
}
