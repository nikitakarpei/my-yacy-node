package indexabstract_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/indexabstract"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
)

func TestIndexAbstractTermsOfTermWithMostPostingsRankByPostingsTheNodeHolds(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := indexabstract.IndexAbstractTermsOf(
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

func TestIndexAbstractTermsOfTermWithMostPostingsBreakTiesBySmallerTerm(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := indexabstract.IndexAbstractTermsOf(
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

func TestIndexAbstractTermsOfTermWithMostPostingsSkipTermsTheNodeDoesNotHold(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := indexabstract.IndexAbstractTermsOf(
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

func TestIndexAbstractTermsOfNoRequestStayEmpty(t *testing.T) {
	if terms := indexabstract.IndexAbstractTermsOf(nil, nil, nil); terms != nil {
		t.Fatalf("terms = %v, want none", terms)
	}
}

func TestIndexAbstractTermsOfTermWithMostPostingsStayEmptyWithoutQueryTerms(t *testing.T) {
	terms := indexabstract.IndexAbstractTermsOf(
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

func TestIndexAbstractTermsOfTermNearestToNodePositionNameTheTermTheDHTSendsHere(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := indexabstract.IndexAbstractTermsOf(
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

func TestIndexAbstractTermsOfTermNearestToNodePositionMeasureForwardAroundTheRing(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := indexabstract.IndexAbstractTermsOf(
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

func TestIndexAbstractTermsOfTermNearestToNodePositionSkipTermsTheNodeDoesNotHold(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := indexabstract.IndexAbstractTermsOf(
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

func TestIndexAbstractTermsOfBothNodeTermsNameEachOfThem(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := indexabstract.IndexAbstractTermsOf(
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

func TestIndexAbstractTermsOfBothNodeTermsCollapseWhenOneTermWinsBothRules(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := indexabstract.IndexAbstractTermsOf(
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

func TestIndexAbstractTermsOfNameTheTermsTheRequestAsksFor(t *testing.T) {
	word, related := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	terms := indexabstract.IndexAbstractTermsOf(
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

func TestIndexAbstractsOfNameTheDocumentsBehindEachTerm(t *testing.T) {
	word, related := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsOf(
		[]yacymodel.Hash{word, related},
		map[yacymodel.Hash][]yacymodel.URLHash{
			word: {searchtest.URLHashFor("u1")},
		},
	)
	if len(abstracts[word]) != 1 {
		t.Fatalf("abstracts = %v, want one document behind w1", abstracts)
	}
	if _, named := abstracts[related]; !named {
		t.Errorf("abstracts = %v, want an empty abstract for w2", abstracts)
	}
}
