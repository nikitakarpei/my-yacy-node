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
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReplicaAsks interface {
	AskForMatchedAndHeldDocuments(
		ctx context.Context,
		asks []peerasks.MatchedAndHeldDocumentsAsk,
	) []peerasks.AnsweredMatchedAndHeldDocumentsAsk
}

type PeerAsks interface {
	AskForCrossCheckedDocuments(
		ctx context.Context,
		asks []peerasks.CrossCheckedDocumentsAsk,
	) []peerasks.AnsweredCrossCheckedDocumentsAsk
	AskForURLMetadata(
		ctx context.Context,
		asks []peerasks.URLMetadataAsk,
	) []peerasks.AnsweredURLMetadataAsk
}

const ListsOnlyTheCrossCheckedDocuments peerjudgements.Question = "lists only the cross-checked documents"

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
	crossCheckedDocumentsCeiling   int
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
	crossCheckedDocumentsCeiling int,
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
		crossCheckedDocumentsCeiling:   crossCheckedDocumentsCeiling,
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

	matchedAndHeldDocumentsRound := spread.askForMatchedAndHeldDocuments(
		ctx,
		query,
		chosenPeersPerQueryWord,
	)
	peerStandings := spread.peerJudgements.StandingsOf(
		ctx, peersAtVersionOf(matchedAndHeldDocumentsRound.replicasThatMayCrossCheck()),
	)
	crossCheckedDocumentsRound := spread.askForCrossCheckedDocuments(
		ctx,
		matchedAndHeldDocumentsRound,
		peerStandings,
	)
	judgedPeers := judgeAskedPeersIn(crossCheckedDocumentsRound)
	spread.peerJudgements.Add(ctx, judgedPeers)
	joinedDocuments := joinedDocumentsFrom(matchedAndHeldDocumentsRound, crossCheckedDocumentsRound)
	urlMetadataRound := spread.askForURLMetadata(ctx, matchedAndHeldDocumentsRound, joinedDocuments)

	spread.observer.WordJoinedSpreadPerformed(ctx, performedWordJoinedSpreadFrom(
		matchedAndHeldDocumentsRound,
		crossCheckedDocumentsRound,
		peerStandings,
		judgedPeers,
		joinedDocuments,
		urlMetadataRound,
		time.Since(startedAt),
	))

	return answeredQueryFrom(
		query, matchedAndHeldDocumentsRound, joinedDocuments, urlMetadataRound,
	)
}

func (spread Spread) askForMatchedAndHeldDocuments(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) matchedAndHeldDocumentsRound {
	asks := matchedAndHeldDocumentsAsksFor(query, chosenPeersPerQueryWord, spread.peerItemsCeiling)
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheMatchedAndHeldDocuments)
	defer endRound()
	answeredAsks := spread.replicaAsks.AskForMatchedAndHeldDocuments(roundContext, asks)

	return matchedAndHeldDocumentsRound{
		queryWords:   query.WordHashes(),
		asks:         asks,
		answeredAsks: answeredAsks,
		queryWordsFewestDocumentsFirst: queryWordsFewestDocumentsFirstFrom(
			query.WordHashes(), chosenPeersPerQueryWord, spread.partitions, answeredAsks,
		),
		compoundWords: compoundWordsAcrossReplicasFrom(
			query.CompoundWords, chosenPeersPerQueryWord, spread.partitions, answeredAsks,
		),
		amountOfPeersPerDocument: amountOfPeersPerDocumentOf(answeredAsks),
	}
}

const (
	roundsLeftAtTheMatchedAndHeldDocuments = 3
	roundsLeftAtTheCrossCheckedDocuments   = 2
	roundsLeftAtTheURLMetadata             = 1
)

func contextOfRound(ctx context.Context, roundsLeft int) (context.Context, context.CancelFunc) {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, time.Until(deadline)/time.Duration(roundsLeft))
}

func peersAtVersionOf(replicas []queryWordOnReplica) []peerjudgements.PeerAtVersion {
	peers := make([]peerjudgements.PeerAtVersion, 0, len(replicas))
	for _, replica := range replicas {
		peers = append(peers, peerjudgements.PeerAtVersion{
			Peer:    replica.peer.Hash,
			Version: replica.versionClaimed(),
		})
	}

	return peers
}

func (spread Spread) askForCrossCheckedDocuments(
	ctx context.Context,
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	peerStandings peerjudgements.PeerStandings,
) crossCheckedDocumentsRound {
	asks := crossCheckedDocumentsAsksFor(
		matchedAndHeldDocumentsRound.partlyListedQueryWordsBesideTheLeadingQueryWord(),
		matchedAndHeldDocumentsRound.documentsOfTheLeadingQueryWordMostListedFirst(),
		peerStandings,
		spread.crossCheckedDocumentsCeiling,
	)
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheCrossCheckedDocuments)
	defer endRound()

	return crossCheckedDocumentsRound{
		asks:         asks,
		answeredAsks: spread.peerAsks.AskForCrossCheckedDocuments(roundContext, asks),
	}
}

func (spread Spread) askForURLMetadata(
	ctx context.Context,
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinedDocuments distinctDocuments,
) urlMetadataRound {
	documentsWithoutMetadata := documentsWithoutMetadataAmong(
		joinedDocuments, matchedAndHeldDocumentsRound.answeredAsks,
	)
	asks := urlMetadataAsksFor(
		matchedAndHeldDocumentsRound.documentsMostListedFirstAmong(documentsWithoutMetadata),
		matchedAndHeldDocumentsRound.answeredAsks,
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
