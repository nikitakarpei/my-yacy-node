package indexabstract_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/indexabstract"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
)

func TestIndexAbstractsOfTermWithMostPostingsCoverTheTermTheNodeHoldsMostPostingsOf(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := termsCoveredByRequest(
		t,
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
		},
		[]yacymodel.Hash{word1, word2},
		map[yacymodel.Hash]int{word1: 9, word2: 3},
	)
	if !slices.Equal(terms, []yacymodel.Hash{word1}) {
		t.Fatalf("terms = %v, want w1, the term this node holds most postings of", terms)
	}
}

func TestIndexAbstractsOfTermWithMostPostingsBreakTiesBySmallerTerm(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := termsCoveredByRequest(
		t,
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
		},
		[]yacymodel.Hash{word1, word2},
		map[yacymodel.Hash]int{word1: 3, word2: 3},
	)
	if !slices.Equal(terms, []yacymodel.Hash{word1}) {
		t.Fatalf("terms = %v, want the tie broken toward w1", terms)
	}
}

func TestIndexAbstractsOfTermWithMostPostingsSkipTermsTheNodeDoesNotHold(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := termsCoveredByRequest(
		t,
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
		},
		[]yacymodel.Hash{word1, word2},
		map[yacymodel.Hash]int{word1: 0, word2: 3},
	)
	if !slices.Equal(terms, []yacymodel.Hash{word2}) {
		t.Fatalf("terms = %v, want only the term the node holds", terms)
	}
}

func TestIndexAbstractsOfNoRequestStayEmpty(t *testing.T) {
	if terms := termsCoveredByRequest(t, nil, nil, nil); terms != nil {
		t.Fatalf("terms = %v, want none", terms)
	}
}

func TestIndexAbstractsOfTermWithMostPostingsStayEmptyWithoutQueryTerms(t *testing.T) {
	terms := termsCoveredByRequest(
		t,
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
		},
		nil,
		nil,
	)
	if terms != nil {
		t.Fatalf("terms = %v, want none", terms)
	}
}

func TestIndexAbstractsOfTermNearestToNodePositionCoverTheTermTheDHTSendsHere(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := termsCoveredByRequest(
		t,
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermNearestToNodePosition{
				NodePosition: yacymodel.DHTRingPositionOf(word2),
			},
		},
		[]yacymodel.Hash{word1, word2},
		map[yacymodel.Hash]int{word1: 9, word2: 3},
	)
	if !slices.Equal(terms, []yacymodel.Hash{word2}) {
		t.Fatalf("terms = %v, want w2, which sits at this node's position", terms)
	}
}

func TestIndexAbstractsOfTermNearestToNodePositionMeasureForwardAroundTheRing(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := termsCoveredByRequest(
		t,
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermNearestToNodePosition{
				NodePosition: yacymodel.DHTRingPositionOf(word1) - 1,
			},
		},
		[]yacymodel.Hash{word1, word2},
		map[yacymodel.Hash]int{word1: 1, word2: 1},
	)
	if !slices.Equal(terms, []yacymodel.Hash{word2}) {
		t.Fatalf("terms = %v, want w2: w1 sits just past this node and wraps the ring", terms)
	}
}

func TestIndexAbstractsOfTermNearestToNodePositionSkipTermsTheNodeDoesNotHold(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := termsCoveredByRequest(
		t,
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermNearestToNodePosition{
				NodePosition: yacymodel.DHTRingPositionOf(word1),
			},
		},
		[]yacymodel.Hash{word1, word2},
		map[yacymodel.Hash]int{word1: 0, word2: 1},
	)
	if !slices.Equal(terms, []yacymodel.Hash{word2}) {
		t.Fatalf("terms = %v, want only the term the node holds", terms)
	}
}

func TestIndexAbstractsOfBothNodeTermsCoverEachOfThem(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := termsCoveredByRequest(
		t,
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
			indexabstract.IndexAbstractOfTermNearestToNodePosition{
				NodePosition: yacymodel.DHTRingPositionOf(word2),
			},
		},
		[]yacymodel.Hash{word1, word2},
		map[yacymodel.Hash]int{word1: 9, word2: 3},
	)
	if !slices.Equal(terms, []yacymodel.Hash{word1, word2}) {
		t.Fatalf("terms = %v, want w1 and w2", terms)
	}
}

func TestIndexAbstractsOfBothNodeTermsCollapseWhenOneTermWinsBothRules(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := termsCoveredByRequest(
		t,
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
			indexabstract.IndexAbstractOfTermNearestToNodePosition{
				NodePosition: yacymodel.DHTRingPositionOf(word1),
			},
		},
		[]yacymodel.Hash{word1, word2},
		map[yacymodel.Hash]int{word1: 9, word2: 3},
	)
	if !slices.Equal(terms, []yacymodel.Hash{word1}) {
		t.Fatalf("terms = %v, want one term for w1", terms)
	}
}

func TestIndexAbstractsOfNamedTermsCoverTheTermsTheRequestAsksFor(t *testing.T) {
	word, related := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := termsCoveredByRequest(
		t,
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractsOfTerms{Terms: []yacymodel.Hash{related}},
		},
		[]yacymodel.Hash{word},
		map[yacymodel.Hash]int{word: 1},
	)
	if !slices.Equal(terms, []yacymodel.Hash{related}) {
		t.Fatalf("terms = %v, want only the terms the request named", terms)
	}
}

func TestIndexAbstractsNameACoveredTermTheNodeHoldsNoDocumentsFor(t *testing.T) {
	word, related := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingOf(word, "u1", 1)},
	}}

	abstracts, err := indexAbstractsFrom(
		t,
		indexabstract.New(index, index, documentsPerIndexAbstract),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word, related}},
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractsOfTerms{Terms: []yacymodel.Hash{word, related}},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("IndexAbstractsFor: %v", err)
	}
	if len(abstracts[word]) != 1 {
		t.Fatalf("abstracts = %v, want one document behind w1", abstracts)
	}
	if _, named := abstracts[related]; !named {
		t.Errorf("abstracts = %v, want an empty abstract for w2", abstracts)
	}
}

func termsCoveredByRequest(
	t *testing.T,
	requested indexabstract.RequestedIndexAbstracts,
	queryTerms []yacymodel.Hash,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) []yacymodel.Hash {
	t.Helper()

	abstracts, err := indexAbstractsFrom(
		t,
		indexabstract.New(
			searchtest.PostingIndex{}, searchtest.PostingIndex{}, documentsPerIndexAbstract,
		),
		searchcriteria.Criteria{Terms: queryTerms},
		requested,
		amountOfPostingsPerTerm,
	)
	if err != nil {
		t.Fatalf("IndexAbstractsFor: %v", err)
	}
	terms := slices.Collect(maps.Keys(abstracts))
	slices.SortFunc(terms, func(a, b yacymodel.Hash) int {
		return yacymodel.CompareInAlphabetOrder(a.String(), b.String())
	})

	return terms
}
