package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func discoveryAsksFor(
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	itemsCeiling int,
) []peerasks.MatchedAndHeldDocumentsAsk {
	var asks []peerasks.MatchedAndHeldDocumentsAsk
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		for _, chosenPeer := range chosenPeersOfQueryWord.ChosenPeers {
			asks = append(asks, peerasks.MatchedAndHeldDocumentsAsk{
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
