package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func matchedAndHeldDocumentsAsksFor(
	query searchquery.Query,
	chosenPeersPerQueryWord [][]peerchoice.ChosenPeer,
	itemsCeiling int,
) []peerasks.MatchedAndHeldDocumentsAsk {
	var asks []peerasks.MatchedAndHeldDocumentsAsk
	for turn := 0; ; turn++ {
		asksInTheTurn := matchedAndHeldDocumentsAsksInTurn(
			turn, query, chosenPeersPerQueryWord, itemsCeiling,
		)
		if len(asksInTheTurn) == 0 {
			return asks
		}
		asks = append(asks, asksInTheTurn...)
	}
}

func matchedAndHeldDocumentsAsksInTurn(
	turn int,
	query searchquery.Query,
	chosenPeersPerQueryWord [][]peerchoice.ChosenPeer,
	itemsCeiling int,
) []peerasks.MatchedAndHeldDocumentsAsk {
	asksInTheTurn := make([]peerasks.MatchedAndHeldDocumentsAsk, 0, len(query.TermHashes()))
	for index, queryWord := range query.TermHashes() {
		chosenPeers := chosenPeersPerQueryWord[index]
		if turn >= len(chosenPeers) {
			continue
		}
		asksInTheTurn = append(asksInTheTurn, peerasks.MatchedAndHeldDocumentsAsk{
			Peer:          chosenPeers[turn].Peer,
			Word:          queryWord,
			ExcludedWords: query.ExclusionHashes(),
			Language:      query.Language,
			ItemsCeiling:  itemsCeiling,
		})
	}

	return asksInTheTurn
}
