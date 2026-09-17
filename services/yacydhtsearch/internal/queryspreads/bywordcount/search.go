// Package bywordcount spreads a query with the word joined spread when the
// query has more than one word, and with the peer matched spread when the query
// has one word.
package bywordcount

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

type QuerySpread interface {
	SpreadOverPeers(
		ctx context.Context,
		query searchquery.Query,
		chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	) queryanswers.AnsweredQuery
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
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) queryanswers.AnsweredQuery {
	if len(query.TermHashes()) < 2 {
		return s.peerMatchedSpread.SpreadOverPeers(ctx, query, chosenPeersPerQueryWord)
	}

	return s.wordJoinedSpread.SpreadOverPeers(ctx, query, chosenPeersPerQueryWord)
}
