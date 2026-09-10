package joinablequery_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/joinablequery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

type countedSpread struct {
	searches int
}

func (c *countedSpread) SpreadOverPeers(
	_ context.Context,
	_ searchquery.Query,
	_ []peerdirectory.AskablePeer,
) [][]searchresult.Item {
	c.searches++

	return nil
}

func searchesOf(t *testing.T, spelledQuery string) (int, int) {
	t.Helper()

	wordJoinedSpread, peerMatchedSpread := &countedSpread{}, &countedSpread{}
	joinablequery.New(wordJoinedSpread, peerMatchedSpread).SpreadOverPeers(
		t.Context(),
		searchquery.QueryFrom(spelledQuery),
		nil,
	)

	return wordJoinedSpread.searches, peerMatchedSpread.searches
}

func TestAQueryWithWordsToJoinGoesToTheWordJoinedSearch(t *testing.T) {
	t.Parallel()

	wordJoined, peerMatched := searchesOf(t, "berlin weather")

	if wordJoined != 1 || peerMatched != 0 {
		t.Fatalf(
			"the word joined search ran %d times and the peer matched search %d, want one and none",
			wordJoined,
			peerMatched,
		)
	}
}

func TestAQueryOfOneWordGoesToThePeerMatchedSearch(t *testing.T) {
	t.Parallel()

	wordJoined, peerMatched := searchesOf(t, "berlin")

	if wordJoined != 0 || peerMatched != 1 {
		t.Fatalf(
			"the word joined search ran %d times and the peer matched search %d, want none and one",
			wordJoined,
			peerMatched,
		)
	}
}

func TestAQueryWhoseWordsRepeatHasNothingToJoin(t *testing.T) {
	t.Parallel()

	wordJoined, peerMatched := searchesOf(t, "berlin berlin")

	if wordJoined != 0 || peerMatched != 1 {
		t.Fatalf(
			"the word joined search ran %d times and the peer matched search %d, want none and one",
			wordJoined,
			peerMatched,
		)
	}
}
