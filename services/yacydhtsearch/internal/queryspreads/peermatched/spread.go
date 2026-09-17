// Package peermatched collects the documents each peer matched for the whole
// query on its own.
package peermatched

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
	) peerchoice.ChosenPeersPerQueryWord
}

type PeerAsks interface {
	AskForMatchedDocuments(
		ctx context.Context,
		asks []peerasks.MatchedDocumentsAsk,
	) []peerasks.AnsweredMatchedDocumentsAsk
}

type Spread struct {
	peerAsks         PeerAsks
	peerChoice       PeerChoice
	peerItemsCeiling int
	observer         PeerMatchedSpreadObserver
}

func New(
	peerAsks PeerAsks,
	peerChoice PeerChoice,
	peerItemsCeiling int,
	observer PeerMatchedSpreadObserver,
) Spread {
	return Spread{
		peerAsks:         peerAsks,
		peerChoice:       peerChoice,
		peerItemsCeiling: peerItemsCeiling,
		observer:         observer,
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
	chosenPeers := chosenPeersPerQueryWord.PeersAcrossQueryWords()
	asks := matchedDocumentsAsksFor(query, chosenPeers, spread.peerItemsCeiling)
	answeredAsks := spread.peerAsks.AskForMatchedDocuments(ctx, asks)

	spread.observer.PeerMatchedSpreadPerformed(
		ctx,
		performedPeerMatchedSpreadFrom(
			query.TermHashes(),
			asks,
			answeredAsks,
			time.Since(startedAt),
		),
	)

	return answeredQueryFrom(answeredAsks, query.TermHashes())
}
