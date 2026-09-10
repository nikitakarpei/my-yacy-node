// Package joinablequery spreads a query with the word joined spread when the
// query has more than one word to join, and with the peer matched spread when
// it has not.
package joinablequery

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

type QuerySpread interface {
	SpreadOverPeers(
		ctx context.Context,
		query searchquery.Query,
		askablePeers []peerdirectory.AskablePeer,
	) [][]searchresult.Item
}

type Spread struct {
	wordJoinedSpread  QuerySpread
	peerMatchedSpread QuerySpread
}

func New(wordJoinedSpread, peerMatchedSpread QuerySpread) Spread {
	return Spread{wordJoinedSpread: wordJoinedSpread, peerMatchedSpread: peerMatchedSpread}
}

func (s Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) [][]searchresult.Item {
	if len(query.TermHashes()) < 2 {
		return s.peerMatchedSpread.SpreadOverPeers(ctx, query, askablePeers)
	}

	return s.wordJoinedSpread.SpreadOverPeers(ctx, query, askablePeers)
}
