// Package wordjoined finds documents that match the whole query when no single
// peer holds every query word, by joining what the peers of each word hold.
package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReplicaAsks interface {
	Start(ctx context.Context) replicaasks.Run
}

type PeerAsks interface {
	AskForSearchDocuments(
		ctx context.Context,
		asks []peerasks.SearchDocumentsAsk,
	) []peerasks.AnsweredSearchDocumentsAsk
	AskForURLMetadata(
		ctx context.Context,
		asks []peerasks.URLMetadataAsk,
	) []peerasks.AnsweredURLMetadataAsk
}

const AbstractHoldsOnlyTheDocumentsToMatch peerjudgements.Question = "abstract holds only the documents to match"

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

	discoveryRound := spread.askToDiscover(
		ctx,
		query,
		chosenPeersPerQueryWord,
	)
	crossCheckCandidates := crossCheckCandidatesIn(discoveryRound, spread.partitions)
	peerStandings := spread.peerJudgements.StandingsOf(
		ctx, peersThatMayCrossCheckIn(crossCheckCandidates),
	)
	crossCheckRound := spread.askToCrossCheck(
		ctx,
		crossCheckCandidates,
		peerStandings,
	)
	judgedPeers := crossCheckJudgementsIn(crossCheckRound)
	spread.peerJudgements.Add(ctx, judgedPeers)
	joinedDocuments := joinedDocumentsFrom(discoveryRound, crossCheckRound)
	urlMetadataLookupRound := spread.askForURLMetadata(ctx, discoveryRound, joinedDocuments)

	spread.observer.WordJoinedSpreadPerformed(ctx, performedWordJoinedSpreadFrom(
		discoveryRound,
		crossCheckRound,
		peerStandings,
		judgedPeers,
		joinedDocuments,
		urlMetadataLookupRound,
		time.Since(startedAt),
	))

	return answeredQueryFrom(
		query, discoveryRound, joinedDocuments, urlMetadataLookupRound,
	)
}

func (spread Spread) askToDiscover(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) discoveryRound {
	asks := discoveryAsksFor(query, chosenPeersPerQueryWord, spread.peerItemsCeiling)
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheDiscovery)
	defer endRound()
	run := spread.replicaAsks.Start(roundContext)
	run.Asks <- asks
	close(run.Asks)
	askOutcomes := askOutcomesFrom(run.SettledWordPartitions)
	answeredAsks := askOutcomes.AnsweredAsks()

	return discoveryRound{
		queryWords:   query.WordHashes(),
		answeredAsks: answeredAsks,
		queryWordsFewestDocumentsFirst: queryWordsFewestDocumentsFirstFrom(
			query.WordHashes(), askOutcomes, spread.partitions,
		),
		compoundWords: compoundWordsAcrossReplicasFrom(
			query.CompoundWords, askOutcomes, spread.partitions,
		),
		holdersPerDocument: holdersPerDocumentOf(answeredAsks),
	}
}

const (
	roundsLeftAtTheDiscovery         = 3
	roundsLeftAtTheCrossCheck        = 2
	roundsLeftAtTheURLMetadataLookup = 1
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
	peerStandings peerjudgements.PeerStandings,
) crossCheckRound {
	asks := crossCheckAsksFor(
		candidates,
		peerStandings,
		spread.documentsToMatchCeiling,
	)
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheCrossCheck)
	defer endRound()

	return crossCheckRound{
		candidates:   candidates,
		asks:         asks,
		answeredAsks: spread.peerAsks.AskForSearchDocuments(roundContext, asks),
	}
}

func (spread Spread) askForURLMetadata(
	ctx context.Context,
	discoveryRound discoveryRound,
	joinedDocuments distinctDocuments,
) urlMetadataLookupRound {
	documentsWithoutMetadata := documentsWithoutMetadataAmong(
		joinedDocuments, discoveryRound.answeredAsks,
	)
	asks := urlMetadataAsksFor(
		discoveryRound.holdersPerDocument.mostHeldFirst(documentsWithoutMetadata),
		discoveryRound.answeredAsks,
		spread.urlMetadataAskDocumentsCeiling,
		spread.amountOfPeersHoldingOneWord,
	)
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheURLMetadataLookup)
	defer endRound()

	return urlMetadataLookupRound{
		documentsWithoutMetadata: documentsWithoutMetadata,
		asks:                     asks,
		answeredAsks:             spread.peerAsks.AskForURLMetadata(roundContext, asks),
	}
}
