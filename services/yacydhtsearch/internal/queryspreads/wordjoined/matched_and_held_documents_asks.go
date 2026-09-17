package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func matchedAndHeldDocumentsAsksFor(
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
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
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	itemsCeiling int,
) []peerasks.MatchedAndHeldDocumentsAsk {
	asksInTheTurn := make([]peerasks.MatchedAndHeldDocumentsAsk, 0, len(chosenPeersPerQueryWord))
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		if turn >= len(chosenPeersOfQueryWord.ChosenPeers) {
			continue
		}
		asksInTheTurn = append(asksInTheTurn, peerasks.MatchedAndHeldDocumentsAsk{
			Peer:          chosenPeersOfQueryWord.ChosenPeers[turn].Peer,
			Word:          chosenPeersOfQueryWord.QueryWord,
			ExcludedWords: query.ExclusionHashes(),
			Language:      query.Language,
			ItemsCeiling:  itemsCeiling,
		})
	}

	return asksInTheTurn
}
