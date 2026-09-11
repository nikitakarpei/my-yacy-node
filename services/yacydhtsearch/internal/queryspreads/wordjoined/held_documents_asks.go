package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func heldDocumentsAsksFor(
	query searchquery.Query,
	chosenPeersPerQueryWord [][]peerdirectory.AskablePeer,
	itemsCeiling int,
) []peerasks.HeldDocumentsAsk {
	var asks []peerasks.HeldDocumentsAsk
	for index, queryWord := range query.TermHashes() {
		for _, peer := range chosenPeersPerQueryWord[index] {
			asks = append(asks, peerasks.HeldDocumentsAsk{
				Peer:          peer,
				Word:          queryWord,
				ExcludedWords: query.ExclusionHashes(),
				Language:      query.Language,
				ItemsCeiling:  itemsCeiling,
			})
		}
	}

	return asks
}
