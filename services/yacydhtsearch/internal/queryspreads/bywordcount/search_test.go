package bywordcount_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/bywordcount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

type countedSpread struct {
	searches int
}

func (c *countedSpread) SpreadOverPeers(
	_ context.Context,
	_ searchquery.Query,
	_ []peerdirectory.AskablePeer,
) peeranswers.AnsweredQuery {
	c.searches++

	return peeranswers.AnsweredQuery{}
}

func searchesOf(t *testing.T, spelledQuery string) (int, int) {
	t.Helper()

	wordJoinedSpread, peerMatchedSpread := &countedSpread{}, &countedSpread{}
	bywordcount.New(wordJoinedSpread, peerMatchedSpread).SpreadOverPeers(
		t.Context(),
		searchquery.QueryFrom(spelledQuery),
		nil,
	)

	return wordJoinedSpread.searches, peerMatchedSpread.searches
}

func TestAQueryOfMoreThanOneWordGoesToTheWordJoinedSpread(t *testing.T) {
	t.Parallel()

	wordJoined, peerMatched := searchesOf(t, "berlin weather")

	if wordJoined != 1 || peerMatched != 0 {
		t.Fatalf(
			"the word joined spread ran %d times and the peer matched spread %d, want one and none",
			wordJoined,
			peerMatched,
		)
	}
}

func TestAQueryOfOneWordGoesToThePeerMatchedSpread(t *testing.T) {
	t.Parallel()

	wordJoined, peerMatched := searchesOf(t, "berlin")

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

	wordJoined, peerMatched := searchesOf(t, "berlin berlin")

	if wordJoined != 0 || peerMatched != 1 {
		t.Fatalf(
			"the word joined spread ran %d times and the peer matched spread %d, want none and one",
			wordJoined,
			peerMatched,
		)
	}
}
