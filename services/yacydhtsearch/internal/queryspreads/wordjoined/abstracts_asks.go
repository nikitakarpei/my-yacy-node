package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func abstractsAsksFor(
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	itemsCeiling int,
) []peerasks.SearchDocumentsAsk {
	var asks []peerasks.SearchDocumentsAsk
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		for _, chosenPeer := range chosenPeersOfQueryWord.ChosenPeers {
			asks = append(asks, peerasks.SearchDocumentsAsk{
				Peer:          chosenPeer.Peer,
				Partition:     chosenPeer.Partition,
				Word:          chosenPeersOfQueryWord.QueryWord,
				ExcludedWords: query.ExclusionHashes(),
				Language:      query.Language,
				ItemsCeiling:  itemsCeiling,
			})
		}
	}

	return asks
}
