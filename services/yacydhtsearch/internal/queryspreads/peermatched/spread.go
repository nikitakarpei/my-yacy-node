// Package peermatched collects what each asked peer matched for the whole
// query on its own. It asks the peers the DHT ring makes responsible for any
// word of the query, up to the peer calls one query may put. Every document a
// peer answers matched every word of the query. A peer counts a word in a
// document without naming the word it counted, so only a query of one word says
// which word the count belongs to.
package peermatched

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
		peerCallsCeiling int,
	) [][]peerdirectory.AskablePeer
}

type PeerAsks interface {
	AskForMatchedItems(
		ctx context.Context,
		asks []peerasks.MatchedItemsAsk,
	) []peerasks.AnsweredMatchedItemsAsk
}

type Spread struct {
	peerAsks         PeerAsks
	peerChoice       PeerChoice
	peerItemsCeiling int
	peerCallsCeiling int
	observer         PeerMatchedSpreadObserver
}

func New(
	peerAsks PeerAsks,
	peerChoice PeerChoice,
	peerItemsCeiling int,
	peerCallsCeiling int,
	observer PeerMatchedSpreadObserver,
) Spread {
	return Spread{
		peerAsks:         peerAsks,
		peerChoice:       peerChoice,
		peerItemsCeiling: peerItemsCeiling,
		peerCallsCeiling: peerCallsCeiling,
		observer:         observer,
	}
}

func (s Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) peeranswers.AnsweredQuery {
	startedAt := time.Now()

	chosenPeersPerQueryWord := s.peerChoice.ChoosePeersPerQueryWord(
		ctx, query.TermHashes(), askablePeers, s.peerCallsCeiling,
	)
	chosenPeers := peersAcrossQueryWords(chosenPeersPerQueryWord, s.peerCallsCeiling)
	asks := matchedItemsAsksFor(query, chosenPeers, s.peerItemsCeiling)
	answeredAsks := s.peerAsks.AskForMatchedItems(ctx, asks)

	s.observer.PeerMatchedSpreadPerformed(
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
