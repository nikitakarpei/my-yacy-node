// Package wordjoined finds documents that match the whole query even when no
// single peer holds every query word. It runs three rounds of peer calls. The
// first round asks the peers of each query word which documents they hold and
// which documents they match for that word. The second round takes the
// documents of the query word with the fewest and asks the peers of every word
// that came back cut off which of those documents they hold. The third round
// asks the peers that hold the joined documents for the metadata the earlier
// rounds did not carry.
package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerChoice interface {
	ChoosePeersPerQueryWord(
		ctx context.Context,
		queryWords []yacymodel.Hash,
		askablePeers []peerdirectory.AskablePeer,
	) [][]peerchoice.ChosenPeer
}

type PeerAsks interface {
	AskForMatchedAndHeldDocuments(
		ctx context.Context,
		asks []peerasks.MatchedAndHeldDocumentsAsk,
	) []peerasks.AnsweredMatchedAndHeldDocumentsAsk
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
	peerAsks                    PeerAsks
	peerChoice                  PeerChoice
	metadataDocumentsCeiling    int
	heldDocumentsCeiling        int
	peerItemsCeiling            int
	partitions                  yacymodel.DHTRingPartitions
	amountOfPeersHoldingOneWord int
	observer                    WordJoinedSpreadObserver
}

//nolint:revive // argument-limit: the ceilings and the ring one word joined spread stays within
func New(
	peerAsks PeerAsks,
	peerChoice PeerChoice,
	metadataDocumentsCeiling int,
	heldDocumentsCeiling int,
	peerItemsCeiling int,
	partitions yacymodel.DHTRingPartitions,
	amountOfPeersHoldingOneWord int,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		peerAsks:                    peerAsks,
		peerChoice:                  peerChoice,
		metadataDocumentsCeiling:    metadataDocumentsCeiling,
		heldDocumentsCeiling:        heldDocumentsCeiling,
		peerItemsCeiling:            peerItemsCeiling,
		partitions:                  partitions,
		amountOfPeersHoldingOneWord: amountOfPeersHoldingOneWord,
		observer:                    observer,
	}
}

// TECHDEBT: vocabulary — searchquery says term, peerasks and the spreads say word, for one fact.
func (spread Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) queryanswers.AnsweredQuery {
	startedAt := time.Now()

	matchedAndHeldDocumentsRound := spread.askForMatchedAndHeldDocuments(ctx, query, askablePeers)
	heldDocumentsRound := spread.askForHeldDocuments(ctx, matchedAndHeldDocumentsRound)
	joinOfTheQuery := joinOfTheQueryFrom(matchedAndHeldDocumentsRound, heldDocumentsRound)
	urlMetadataRound := spread.askForURLMetadata(ctx, matchedAndHeldDocumentsRound, joinOfTheQuery)

	spread.observer.WordJoinedSpreadPerformed(ctx, performedWordJoinedSpreadFrom(
		matchedAndHeldDocumentsRound,
		heldDocumentsRound,
		joinOfTheQuery,
		urlMetadataRound,
		time.Since(startedAt),
	))

	return answeredQueryFrom(matchedAndHeldDocumentsRound, joinOfTheQuery, urlMetadataRound)
}

func (spread Spread) askForMatchedAndHeldDocuments(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) matchedAndHeldDocumentsRound {
	chosenPeersPerQueryWord := spread.peerChoice.ChoosePeersPerQueryWord(
		ctx, query.TermHashes(), askablePeers,
	)
	asks := matchedAndHeldDocumentsAsksFor(query, chosenPeersPerQueryWord, spread.peerItemsCeiling)
	roundContext, endRound := contextOfRound(ctx, amountOfRoundsOfPeerCalls)
	defer endRound()
	answeredAsks := spread.peerAsks.AskForMatchedAndHeldDocuments(roundContext, asks)

	return matchedAndHeldDocumentsRound{
		queryWords:   query.TermHashes(),
		asks:         asks,
		answeredAsks: answeredAsks,
		queryWordsFewestDocumentsFirst: queryWordsFewestDocumentsFirstFrom(
			query.TermHashes(), spread.partitions, chosenPeersPerQueryWord, answeredAsks,
		),
		amountOfPeersPerDocument: amountOfPeersPerDocumentOf(answeredAsks),
	}
}

const (
	amountOfRoundsOfPeerCalls    = 3
	roundsLeftAtTheHeldDocuments = 2
	roundsLeftAtTheURLMetadata   = 1
)

func contextOfRound(ctx context.Context, roundsLeft int) (context.Context, context.CancelFunc) {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, time.Until(deadline)/time.Duration(roundsLeft))
}

func (spread Spread) askForHeldDocuments(
	ctx context.Context,
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
) heldDocumentsRound {
	deal := heldDocumentsDealFor(
		matchedAndHeldDocumentsRound.cutOffQueryWords(),
		matchedAndHeldDocumentsRound.documentsMostHeldFirstAmong(
			matchedAndHeldDocumentsRound.anchor().documentsHeld(),
		),
		spread.heldDocumentsCeiling,
	)
	roundContext, endRound := contextOfRound(ctx, roundsLeftAtTheHeldDocuments)
	defer endRound()

	return heldDocumentsRound{
		asks:         deal.asks,
		answeredAsks: spread.peerAsks.AskForHeldDocuments(roundContext, deal.asks),
		amountOfDocumentsPastTheHeldDocumentsCeiling: len(deal.documentsPastTheCeiling),
	}
}

func (spread Spread) askForURLMetadata(
	ctx context.Context,
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinOfTheQuery joinOfTheQuery,
) urlMetadataRound {
	documentsWithoutMetadata := documentsWithoutMetadataAmong(
		joinOfTheQuery.joinedDocuments, matchedAndHeldDocumentsRound.answeredAsks,
	)
	asks := urlMetadataAsksFor(
		matchedAndHeldDocumentsRound.documentsMostHeldFirstAmong(documentsWithoutMetadata),
		matchedAndHeldDocumentsRound.answeredAsks,
		spread.metadataDocumentsCeiling,
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
