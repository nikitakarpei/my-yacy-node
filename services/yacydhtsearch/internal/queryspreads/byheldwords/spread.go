// Package byheldwords spreads a query with the peer matched spread when every
// chosen peer holds every query word, and with the word joined spread when it
// does not. Every peer holds every word when the network has no more peers
// than the network redundancy.
package byheldwords

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
	networkRedundancy int
}

func New(wordJoinedSpread, peerMatchedSpread QuerySpread, networkRedundancy int) Spread {
	return Spread{
		wordJoinedSpread:  wordJoinedSpread,
		peerMatchedSpread: peerMatchedSpread,
		networkRedundancy: networkRedundancy,
	}
}

func (s Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) queryanswers.AnsweredQuery {
	if s.peersHoldTheWholeQuery(query, chosenPeersPerQueryWord) {
		return s.peerMatchedSpread.SpreadOverPeers(ctx, query, chosenPeersPerQueryWord)
	}

	return s.wordJoinedSpread.SpreadOverPeers(ctx, query, chosenPeersPerQueryWord)
}

func (s Spread) peersHoldTheWholeQuery(
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) bool {
	return len(query.WordHashes()) < 2 ||
		len(chosenPeersPerQueryWord.ChosenPeersAcrossQueryWords()) <= s.networkRedundancy
}
