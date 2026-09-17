// Package wordjoined finds documents that match the whole query even when no
// single peer holds every query word. It runs three rounds of peer calls. The
// first round asks the peers of each query word which documents they hold and
// which documents they match for that word. The second round takes the
// documents of the query word with the fewest and asks the peers of every word
// that answered short which of those documents they hold. The third round asks
// the peers that hold the joined documents for the metadata the earlier rounds
// did not carry.
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
	) []peerchoice.PeersOfQueryWord
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
	amountOfPeersHoldingOneWord int
	observer                    WordJoinedSpreadObserver
}

//nolint:revive // argument-limit: the ceilings one word joined spread stays within
func New(
	peerAsks PeerAsks,
	peerChoice PeerChoice,
	metadataDocumentsCeiling int,
	heldDocumentsCeiling int,
	peerItemsCeiling int,
	amountOfPeersHoldingOneWord int,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		peerAsks:                    peerAsks,
		peerChoice:                  peerChoice,
		metadataDocumentsCeiling:    metadataDocumentsCeiling,
		heldDocumentsCeiling:        heldDocumentsCeiling,
		peerItemsCeiling:            peerItemsCeiling,
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

	chosenPeersPerQueryWord := spread.peerChoice.ChoosePeersPerQueryWord(
		ctx, query.TermHashes(), askablePeers,
	)
	matchedAndHeldDocumentsAsks, answeredMatchedAndHeldDocumentsAsks := spread.askForMatchedAndHeldDocuments(
		ctx,
		query,
		chosenPeersPerQueryWord,
	)
	amountOfDocumentsHeldPerQueryWord := amountOfDocumentsHeldPerQueryWordOf(
		answeredMatchedAndHeldDocumentsAsks, chosenPeersPerQueryWord, query.TermHashes(),
	)
	anchor := anchorOfTheQueryFrom(
		query.TermHashes(),
		answeredMatchedAndHeldDocumentsAsks,
		amountOfDocumentsHeldPerQueryWord,
	)
	shortQueryWords := shortQueryWordsOf(
		query.TermHashes(),
		chosenPeersPerQueryWord,
		answeredMatchedAndHeldDocumentsAsks,
		amountOfDocumentsHeldPerQueryWord,
		anchor.word,
	)
	heldDocuments := spread.askForHeldDocuments(
		ctx, anchor, shortQueryWords, answeredMatchedAndHeldDocumentsAsks,
	)
	joinedDocuments := joinedDocumentsOf(
		anchor,
		answeredMatchedAndHeldDocumentsAsks,
		heldDocuments.answeredAsks,
		query.TermHashes(),
	)
	itemsInTheOrderOfEachPeerRanking := itemsInTheOrderOfEachPeerRankingOf(
		answeredMatchedAndHeldDocumentsAsks, joinedDocuments,
	)
	documentsWithoutMetadata := joinedDocumentsWithoutMetadata(
		joinedDocuments, itemsInTheOrderOfEachPeerRanking,
	)
	urlMetadataAsks, answeredURLMetadataAsks := spread.askForURLMetadata(
		ctx, documentsWithoutMetadata, answeredMatchedAndHeldDocumentsAsks,
	)

	spread.observer.WordJoinedSpreadPerformed(
		ctx,
		performedWordJoinedSpreadFrom(
			query.TermHashes(),
			matchedAndHeldDocumentsAsks,
			answeredMatchedAndHeldDocumentsAsks,
			anchor,
			heldDocuments,
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
		amountOfDocumentsHeldPerQueryWord,
		query.TermHashes(),
	)
}

func (spread Spread) askForMatchedAndHeldDocuments(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord []peerchoice.PeersOfQueryWord,
) ([]peerasks.MatchedAndHeldDocumentsAsk, []peerasks.AnsweredMatchedAndHeldDocumentsAsk) {
	asks := matchedAndHeldDocumentsAsksFor(query, chosenPeersPerQueryWord, spread.peerItemsCeiling)
	round, endRound := contextOfRound(ctx, amountOfRoundsOfPeerCalls)
	defer endRound()

	return asks, spread.peerAsks.AskForMatchedAndHeldDocuments(round, asks)
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

type heldDocumentsRound struct {
	amountOfShortQueryWords                      int
	asks                                         []peerasks.HeldDocumentsAsk
	answeredAsks                                 []peerasks.AnsweredHeldDocumentsAsk
	amountOfDocumentsPastTheHeldDocumentsCeiling int
}

func (spread Spread) askForHeldDocuments(
	ctx context.Context,
	anchor anchorOfTheQuery,
	shortQueryWords []shortQueryWord,
	answeredMatchedAndHeldDocumentsAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) heldDocumentsRound {
	asks, amountOfDocumentsPastTheCeiling := heldDocumentsAsksFor(
		shortQueryWords,
		anchor.documents,
		answeredMatchedAndHeldDocumentsAsks,
		spread.heldDocumentsCeiling,
	)
	round, endRound := contextOfRound(ctx, roundsLeftAtTheHeldDocuments)
	defer endRound()

	return heldDocumentsRound{
		amountOfShortQueryWords: len(shortQueryWords),
		asks:                    asks,
		answeredAsks:            spread.peerAsks.AskForHeldDocuments(round, asks),
		amountOfDocumentsPastTheHeldDocumentsCeiling: amountOfDocumentsPastTheCeiling,
	}
}

func (spread Spread) askForURLMetadata(
	ctx context.Context,
	documentsWithoutMetadata map[yacymodel.URLHash]struct{},
	answeredMatchedAndHeldDocumentsAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) ([]peerasks.URLMetadataAsk, []peerasks.AnsweredURLMetadataAsk) {
	asks := urlMetadataAsksFor(
		documentsWithoutMetadata,
		answeredMatchedAndHeldDocumentsAsks,
		spread.metadataDocumentsCeiling,
		spread.amountOfPeersHoldingOneWord,
	)
	round, endRound := contextOfRound(ctx, roundsLeftAtTheURLMetadata)
	defer endRound()

	return asks, spread.peerAsks.AskForURLMetadata(round, asks)
}
