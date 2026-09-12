package indexabstract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/indexabstract"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

func matchOf(urls ...string) termpostings.Match {
	byDocument := make(map[yacymodel.URLHash]yacymodel.RWIPosting, len(urls))
	for _, url := range urls {
		byDocument[searchtest.URLHashFor(url)] = yacymodel.RWIPosting{
			URLHash: searchtest.URLHashFor(url),
		}
	}

	return termpostings.Match{PostingPerDocument: byDocument, PostingsHeld: len(urls)}
}

func matchHolding(postingsHeld int, urls ...string) termpostings.Match {
	match := matchOf(urls...)
	match.PostingsHeld = postingsHeld

	return match
}

func TestIndexAbstractsForTermWithMostPostingsRanksByPostingsTheNodeHolds(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
		},
		map[yacymodel.Hash]termpostings.Match{
			word1: matchHolding(9, "u1"),
			word2: matchHolding(3, "u2", "u3"),
		},
		nil,
	)
	if len(abstracts[word1]) != 1 {
		t.Fatalf("abstracts = %v, want w1, whose postings the cap held back", abstracts)
	}
	if _, ok := abstracts[word2]; ok {
		t.Errorf("abstracts = %v, want only w1", abstracts)
	}
}

func TestIndexAbstractsForTermWithMostPostingsBreakTiesBySmallerTerm(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
		},
		map[yacymodel.Hash]termpostings.Match{
			word1: matchOf("u1"),
			word2: matchOf("u2"),
		},
		nil,
	)
	if _, ok := abstracts[word1]; !ok {
		t.Fatalf("abstracts = %v, want the tie broken toward w1", abstracts)
	}
	if _, ok := abstracts[word2]; ok {
		t.Errorf("abstracts = %v, want only w1", abstracts)
	}
}

func TestIndexAbstractsForTermWithMostPostingsSkipTermsWithoutDocuments(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
		},
		map[yacymodel.Hash]termpostings.Match{
			word1: matchHolding(9),
			word2: matchHolding(3, "u1"),
		},
		nil,
	)
	if _, ok := abstracts[word1]; ok {
		t.Fatalf("abstracts = %v, want no abstract for the term without documents", abstracts)
	}
	if len(abstracts[word2]) != 1 {
		t.Errorf("abstracts = %v, want w2", abstracts)
	}
}

func TestIndexAbstractsForNoRequestStayEmpty(t *testing.T) {
	abstracts := indexabstract.IndexAbstractsFor(nil, nil, nil)
	if abstracts != nil {
		t.Fatalf("abstracts = %v, want none", abstracts)
	}
}

func TestIndexAbstractsForTermWithMostPostingsStayEmptyWithoutMatches(t *testing.T) {
	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
		},
		nil,
		nil,
	)
	if abstracts != nil {
		t.Fatalf("abstracts = %v, want none", abstracts)
	}
}

func TestIndexAbstractsForTermNearestToNodePositionNameTheTermTheDHTSendsHere(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermNearestToNodePosition{
				NodePosition: yacymodel.DHTRingPositionOf(word2),
			},
		},
		map[yacymodel.Hash]termpostings.Match{
			word1: matchHolding(9, "u1"),
			word2: matchHolding(3, "u2"),
		},
		nil,
	)
	if _, ok := abstracts[word2]; !ok {
		t.Fatalf("abstracts = %v, want w2, which sits at this node's position", abstracts)
	}
	if _, ok := abstracts[word1]; ok {
		t.Errorf("abstracts = %v, want only w2", abstracts)
	}
}

func TestIndexAbstractsForTermNearestToNodePositionMeasureForwardAroundTheRing(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermNearestToNodePosition{
				NodePosition: yacymodel.DHTRingPositionOf(word1) - 1,
			},
		},
		map[yacymodel.Hash]termpostings.Match{
			word1: matchOf("u1"),
			word2: matchOf("u2"),
		},
		nil,
	)
	if _, ok := abstracts[word2]; !ok {
		t.Fatalf(
			"abstracts = %v, want w2: w1 sits just past this node and wraps the ring",
			abstracts,
		)
	}
	if _, ok := abstracts[word1]; ok {
		t.Errorf("abstracts = %v, want only w2", abstracts)
	}
}

func TestIndexAbstractsForTermNearestToNodePositionSkipTermsWithoutDocuments(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermNearestToNodePosition{
				NodePosition: yacymodel.DHTRingPositionOf(word1),
			},
		},
		map[yacymodel.Hash]termpostings.Match{
			word1: matchHolding(9),
			word2: matchOf("u1"),
		},
		nil,
	)
	if _, ok := abstracts[word1]; ok {
		t.Fatalf("abstracts = %v, want no abstract for the term without documents", abstracts)
	}
	if len(abstracts[word2]) != 1 {
		t.Errorf("abstracts = %v, want w2", abstracts)
	}
}

func TestIndexAbstractsForBothNodeTermsNameTheDocumentsBehindEach(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
			indexabstract.IndexAbstractOfTermNearestToNodePosition{
				NodePosition: yacymodel.DHTRingPositionOf(word2),
			},
		},
		map[yacymodel.Hash]termpostings.Match{
			word1: matchHolding(9, "u1"),
			word2: matchHolding(3, "u2", "u3"),
		},
		nil,
	)
	if len(abstracts[word1]) != 1 || len(abstracts[word2]) != 2 {
		t.Fatalf("abstracts = %v, want w1 with one document and w2 with two", abstracts)
	}
}

func TestIndexAbstractsForBothNodeTermsCollapseWhenOneTermWinsBothRules(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
			indexabstract.IndexAbstractOfTermNearestToNodePosition{
				NodePosition: yacymodel.DHTRingPositionOf(word1),
			},
		},
		map[yacymodel.Hash]termpostings.Match{
			word1: matchHolding(9, "u1"),
			word2: matchHolding(3, "u2"),
		},
		nil,
	)
	if len(abstracts) != 1 {
		t.Fatalf("abstracts = %v, want one abstract for w1", abstracts)
	}
	if len(abstracts[word1]) != 1 {
		t.Errorf("abstracts = %v, want w1", abstracts)
	}
}

func TestIndexAbstractsOfTermsListTheDocumentsBehindThem(t *testing.T) {
	word, related := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractsOfTerms{Terms: []yacymodel.Hash{related}},
		},
		map[yacymodel.Hash]termpostings.Match{word: matchOf("u1")},
		map[yacymodel.Hash]termpostings.Match{related: matchOf("u2", "u3")},
	)
	if len(abstracts[related]) != 2 {
		t.Fatalf("abstracts = %v, want the two documents behind w2", abstracts)
	}
	if _, ok := abstracts[word]; ok {
		t.Errorf("abstracts = %v, want only the terms the request named", abstracts)
	}
}

func TestIndexAbstractsOfTermsNameTermsWithoutDocuments(t *testing.T) {
	related := searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractsOfTerms{Terms: []yacymodel.Hash{related}},
		},
		nil,
		map[yacymodel.Hash]termpostings.Match{related: matchOf()},
	)
	if _, ok := abstracts[related]; !ok {
		t.Fatalf("abstracts = %v, want an empty abstract for the term the request named", abstracts)
	}
}

func TestIndexAbstractTermsOfNamesOnlyTheTermsAnAbstractOfTermsCovers(t *testing.T) {
	related := searchtest.HashFor("w2")

	if len(indexabstract.IndexAbstractTermsOf(
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractsOfTerms{Terms: []yacymodel.Hash{related}},
		},
	)) != 1 {
		t.Error("IndexAbstractTermsOf named no terms, want w2")
	}
	quiet := indexabstract.RequestedIndexAbstracts{
		indexabstract.IndexAbstractOfTermWithMostPostings{},
		indexabstract.IndexAbstractOfTermNearestToNodePosition{},
	}
	if terms := indexabstract.IndexAbstractTermsOf(quiet); terms != nil {
		t.Errorf("IndexAbstractTermsOf(%v) = %v, want none", quiet, terms)
	}
}
