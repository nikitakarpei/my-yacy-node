package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func matchedDocumentsAsksFor(
	query searchquery.Query,
	chosenPeers []peerchoice.ChosenPeer,
	peerItemsCeiling int,
) []peerasks.MatchedDocumentsAsk {
	asks := make([]peerasks.MatchedDocumentsAsk, 0, len(chosenPeers))
	for _, chosenPeer := range chosenPeers {
		asks = append(asks, peerasks.MatchedDocumentsAsk{
			Peer:          chosenPeer.Peer,
			Partition:     chosenPeer.Partition,
			WordsToMatch:  query.WordHashes(),
			ExcludedWords: query.ExclusionHashes(),
			Language:      query.Language,
			ItemsCeiling:  peerItemsCeiling,
		})
	}

	return asks
}
