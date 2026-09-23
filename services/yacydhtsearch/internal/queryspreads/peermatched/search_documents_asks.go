package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func searchDocumentsAsksFor(
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	peerItemsCeiling int,
) []peerasks.SearchDocumentsAsk {
	queryWords := query.WordHashes()
	chosenPeers := chosenPeersPerQueryWord.ChosenPeersOf(queryWords[0])
	asks := make([]peerasks.SearchDocumentsAsk, 0, len(chosenPeers))
	for _, chosenPeer := range chosenPeers {
		asks = append(asks, peerasks.SearchDocumentsAsk{
			Peer:              chosenPeer.Peer,
			Partition:         chosenPeer.Partition,
			Word:              queryWords[0],
			OtherWordsToMatch: queryWords[1:],
			ExcludedWords:     query.ExclusionHashes(),
			Language:          query.Language,
			ItemsCeiling:      peerItemsCeiling,
		})
	}

	return asks
}
