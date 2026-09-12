// Package wordjoined finds documents that match the whole query even when no
// single peer holds every query word. It asks each peer which documents it
// holds for one query word and which documents it matches for that word, keeps
// the documents that came back for every word, and asks the peers that hold
// them for the metadata of the joined documents that came back without it.
package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerChoice interface {
	ChoosePeersPerQueryWord(
		ctx context.Context,
		queryWords []yacymodel.Hash,
		askablePeers []peerdirectory.AskablePeer,
		amountOfPeersAskedPerWord int,
	) [][]peerdirectory.AskablePeer
}

type PeerAsks interface {
	AskForHeldDocuments(
		ctx context.Context,
		asks []peerasks.HeldDocumentsAsk,
	) []peerasks.AnsweredHeldDocumentsAsk
	AskForURLMetadata(
		ctx context.Context,
		asks []peerasks.URLMetadataAsk,
	) []peerasks.AnsweredURLMetadataAsk
}

type Spread struct {
	peerAsks                  PeerAsks
	peerChoice                PeerChoice
	metadataDocumentsCeiling  int
	peerItemsCeiling          int
	amountOfPeersAskedPerWord int
	observer                  WordJoinedSpreadObserver
}

//nolint:revive // argument-limit: the ceilings one word joined spread stays within
func New(
	peerAsks PeerAsks,
	peerChoice PeerChoice,
	metadataDocumentsCeiling int,
	peerItemsCeiling int,
	amountOfPeersAskedPerWord int,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		peerAsks:                  peerAsks,
		peerChoice:                peerChoice,
		metadataDocumentsCeiling:  metadataDocumentsCeiling,
		peerItemsCeiling:          peerItemsCeiling,
		amountOfPeersAskedPerWord: amountOfPeersAskedPerWord,
		observer:                  observer,
	}
}

// TECHDEBT: vocabulary — searchquery says term, peerasks and the spreads say word, for one fact.
func (spread Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) peeranswers.AnsweredQuery {
	startedAt := time.Now()

	chosenPeersPerQueryWord := spread.peerChoice.ChoosePeersPerQueryWord(
		ctx, query.TermHashes(), askablePeers, spread.amountOfPeersAskedPerWord,
	)
	heldDocumentsAsks, answeredHeldDocumentsAsks := spread.askForHeldDocuments(
		ctx, query, chosenPeersPerQueryWord,
	)
	joinedDocuments := joinedDocumentsOf(answeredHeldDocumentsAsks, query.TermHashes())
	itemsInTheOrderOfEachPeerRanking := itemsInTheOrderOfEachPeerRankingOf(
		answeredHeldDocumentsAsks, joinedDocuments,
	)
	documentsWithoutMetadata := joinedDocumentsWithoutMetadata(
		joinedDocuments, itemsInTheOrderOfEachPeerRanking,
	)
	urlMetadataAsks, answeredURLMetadataAsks := spread.askForURLMetadata(
		ctx, documentsWithoutMetadata, answeredHeldDocumentsAsks,
	)

	spread.observer.WordJoinedSpreadPerformed(
		ctx,
		performedWordJoinedSpreadFrom(
			query.TermHashes(),
			heldDocumentsAsks,
			answeredHeldDocumentsAsks,
			joinedDocuments,
			documentsWithoutMetadata,
			urlMetadataAsks,
			answeredURLMetadataAsks,
			time.Since(startedAt),
		),
	)

	return answeredQueryFrom(
		itemsInTheOrderOfEachPeerRanking,
		answeredURLMetadataAsks,
		answeredHeldDocumentsAsks,
		query.TermHashes(),
	)
}

func (spread Spread) askForHeldDocuments(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord [][]peerdirectory.AskablePeer,
) ([]peerasks.HeldDocumentsAsk, []peerasks.AnsweredHeldDocumentsAsk) {
	asks := heldDocumentsAsksFor(query, chosenPeersPerQueryWord, spread.peerItemsCeiling)
	firstRound, endFirstRound := contextOfTheFirstRound(ctx)
	defer endFirstRound()

	return asks, spread.peerAsks.AskForHeldDocuments(firstRound, asks)
}

const amountOfRoundsOfPeerCalls = 2

func contextOfTheFirstRound(ctx context.Context) (context.Context, context.CancelFunc) {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, time.Until(deadline)/amountOfRoundsOfPeerCalls)
}

func (spread Spread) askForURLMetadata(
	ctx context.Context,
	documentsWithoutMetadata map[yacymodel.URLHash]struct{},
	answeredHeldDocumentsAsks []peerasks.AnsweredHeldDocumentsAsk,
) ([]peerasks.URLMetadataAsk, []peerasks.AnsweredURLMetadataAsk) {
	asks := urlMetadataAsksFor(
		documentsWithoutMetadata,
		answeredHeldDocumentsAsks,
		spread.metadataDocumentsCeiling,
		spread.amountOfPeersAskedPerWord,
	)
	secondRound, endSecondRound := contextOfTheSecondRound(ctx)
	defer endSecondRound()

	return asks, spread.peerAsks.AskForURLMetadata(secondRound, asks)
}

func contextOfTheSecondRound(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}
