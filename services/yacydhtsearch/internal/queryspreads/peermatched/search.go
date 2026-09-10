// Package peermatched collects what every asked peer matched for the whole
// query on its own, which is how a YaCy peer answers a remote search. It asks
// every peer the DHT ring makes responsible for any word of the query.
package peermatched

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
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
	itemsCeiling     int
	peerCallsCeiling int
	observer         PeerMatchedSearchObserver
}

func New(
	peerAsks PeerAsks,
	peerChoice PeerChoice,
	itemsCeiling int,
	peerCallsCeiling int,
	observer PeerMatchedSearchObserver,
) Spread {
	return Spread{
		peerAsks:         peerAsks,
		peerChoice:       peerChoice,
		itemsCeiling:     itemsCeiling,
		peerCallsCeiling: peerCallsCeiling,
		observer:         observer,
	}
}

func (s Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) [][]searchresult.Item {
	startedAt := time.Now()

	chosenPeers := s.choosePeersForTheWholeQuery(ctx, query, askablePeers)
	asks := asksForTheWholeQuery(query, chosenPeers, s.itemsCeiling)
	answeredAsks := s.peerAsks.AskForMatchedItems(ctx, asks)

	s.observer.PeerMatchedSearchPerformed(
		ctx,
		performedPeerMatchedSearchFrom(
			query.TermHashes(),
			asks,
			answeredAsks,
			time.Since(startedAt),
		),
	)

	return itemsOfEachAnsweredAsk(answeredAsks)
}

func (s Spread) choosePeersForTheWholeQuery(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	peersPerQueryWord := s.peerChoice.ChoosePeersPerQueryWord(
		ctx, query.TermHashes(), askablePeers, s.peerCallsCeiling,
	)

	chosenPeers := make([]peerdirectory.AskablePeer, 0, s.peerCallsCeiling)
	takenPeers := map[yacymodel.Hash]struct{}{}
	for _, peersOfQueryWord := range peersPerQueryWord {
		for _, peer := range peersOfQueryWord {
			if _, taken := takenPeers[peer.Hash]; taken {
				continue
			}
			if len(chosenPeers) == s.peerCallsCeiling {
				return chosenPeers
			}
			takenPeers[peer.Hash] = struct{}{}
			chosenPeers = append(chosenPeers, peer)
		}
	}

	return chosenPeers
}

func asksForTheWholeQuery(
	query searchquery.Query,
	chosenPeers []peerdirectory.AskablePeer,
	itemsCeiling int,
) []peerasks.MatchedItemsAsk {
	asks := make([]peerasks.MatchedItemsAsk, 0, len(chosenPeers))
	for _, peer := range chosenPeers {
		asks = append(asks, peerasks.MatchedItemsAsk{
			Peer:          peer,
			WordsToMatch:  query.TermHashes(),
			ExcludedWords: query.ExclusionHashes(),
			Language:      query.Language,
			ItemsCeiling:  itemsCeiling,
		})
	}

	return asks
}

func itemsOfEachAnsweredAsk(
	answeredAsks []peerasks.AnsweredMatchedItemsAsk,
) [][]searchresult.Item {
	items := make([][]searchresult.Item, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		items = append(items, answeredAsk.Items)
	}

	return items
}
