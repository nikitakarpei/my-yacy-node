package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func matchedDocumentsAsksFor(
	query searchquery.Query,
	chosenPeers []peerdirectory.AskablePeer,
	peerItemsCeiling int,
) []peerasks.MatchedDocumentsAsk {
	asks := make([]peerasks.MatchedDocumentsAsk, 0, len(chosenPeers))
	for _, peer := range chosenPeers {
		asks = append(asks, peerasks.MatchedDocumentsAsk{
			Peer:          peer,
			WordsToMatch:  query.TermHashes(),
			ExcludedWords: query.ExclusionHashes(),
			Language:      query.Language,
			ItemsCeiling:  peerItemsCeiling,
		})
	}

	return asks
}
