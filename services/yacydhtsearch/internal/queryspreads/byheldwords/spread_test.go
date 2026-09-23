package byheldwords_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/byheldwords"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const networkRedundancy = 2

type countedSpread struct {
	searches int
}

func (c *countedSpread) SpreadOverPeers(
	_ context.Context,
	_ searchquery.Query,
	_ peerchoice.ChosenPeersPerQueryWord,
) queryanswers.AnsweredQuery {
	c.searches++

	return queryanswers.AnsweredQuery{}
}

func searchesOf(t *testing.T, spelledQuery string, peerNames ...string) (int, int) {
	t.Helper()

	query := searchquery.QueryFrom(spelledQuery, "")
	wordJoinedSpread, peerMatchedSpread := &countedSpread{}, &countedSpread{}
	byheldwords.New(wordJoinedSpread, peerMatchedSpread, networkRedundancy).SpreadOverPeers(
		t.Context(),
		query,
		everyPeerChosenForEachWordOf(query, peerNames),
	)

	return wordJoinedSpread.searches, peerMatchedSpread.searches
}

func everyPeerChosenForEachWordOf(
	query searchquery.Query,
	peerNames []string,
) peerchoice.ChosenPeersPerQueryWord {
	chosenPeers := make([]peerchoice.ChosenPeer, 0, len(peerNames))
	for _, peerName := range peerNames {
		chosenPeers = append(chosenPeers, peerchoice.ChosenPeer{
			Peer: peerdirectory.AskablePeer{Hash: yacymodel.WordHash(peerName)},
		})
	}
	chosenPeersPerQueryWord := make(peerchoice.ChosenPeersPerQueryWord, 0, len(query.WordHashes()))
	for _, queryWord := range query.WordHashes() {
		chosenPeersPerQueryWord = append(chosenPeersPerQueryWord, peerchoice.ChosenPeersOfQueryWord{
			QueryWord:   queryWord,
			ChosenPeers: chosenPeers,
		})
	}

	return chosenPeersPerQueryWord
}

func TestAQueryOfMoreThanOneWordOverMorePeersThanTheRedundancyGoesToTheWordJoinedSpread(
	t *testing.T,
) {
	t.Parallel()

	wordJoined, peerMatched := searchesOf(t, "berlin weather", "first", "second", "third")

	if wordJoined != 1 || peerMatched != 0 {
		t.Fatalf(
			"the word joined spread ran %d times and the peer matched spread %d, want one and none",
			wordJoined,
			peerMatched,
		)
	}
}

func TestAQueryOfMoreThanOneWordOverNoMorePeersThanTheRedundancyGoesToThePeerMatchedSpread(
	t *testing.T,
) {
	t.Parallel()

	wordJoined, peerMatched := searchesOf(t, "berlin weather", "first", "second")

	if wordJoined != 0 || peerMatched != 1 {
		t.Fatalf(
			"the word joined spread ran %d times and the peer matched spread %d, want none and one",
			wordJoined,
			peerMatched,
		)
	}
}

func TestAQueryOfOneWordGoesToThePeerMatchedSpread(t *testing.T) {
	t.Parallel()

	wordJoined, peerMatched := searchesOf(t, "berlin", "first", "second", "third")

	if wordJoined != 0 || peerMatched != 1 {
		t.Fatalf(
			"the word joined spread ran %d times and the peer matched spread %d, want none and one",
			wordJoined,
			peerMatched,
		)
	}
}

func TestAQueryWhoseWordsRepeatCountsAsOneWord(t *testing.T) {
	t.Parallel()

	wordJoined, peerMatched := searchesOf(t, "berlin berlin", "first", "second", "third")

	if wordJoined != 0 || peerMatched != 1 {
		t.Fatalf(
			"the word joined spread ran %d times and the peer matched spread %d, want none and one",
			wordJoined,
			peerMatched,
		)
	}
}
