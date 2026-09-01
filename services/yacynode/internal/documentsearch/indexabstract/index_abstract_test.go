package indexabstract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/indexabstract"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

func matchOf(urls ...string) termpostings.Match {
	byDocument := make(map[yacymodel.URLHash]termpostings.Posting, len(urls))
	for _, url := range urls {
		byDocument[searchtest.URLHashFor(url)] = termpostings.Posting{
			DocumentHash: searchtest.URLHashFor(url),
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
		indexabstract.IndexAbstractOfTermWithMostPostings{},
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word1, word2}},
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

func TestIndexAbstractsForNoIndexAbstractStayEmpty(t *testing.T) {
	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.NoIndexAbstracts{},
		searchcriteria.Criteria{},
		nil,
		nil,
	)
	if abstracts != nil {
		t.Fatalf("abstracts = %v, want none", abstracts)
	}
}

func TestIndexAbstractsForTermWithMostPostingsStayEmptyOnSingleTerm(t *testing.T) {
	word := searchtest.HashFor("w1")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.IndexAbstractOfTermWithMostPostings{},
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}},
		map[yacymodel.Hash]termpostings.Match{word: matchOf("u1")},
		nil,
	)
	if abstracts != nil {
		t.Fatalf("abstracts = %v, want none", abstracts)
	}
}

func TestIndexAbstractsForTermWithMostPostingsStayEmptyWithRequiredDocuments(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.IndexAbstractOfTermWithMostPostings{},
		searchcriteria.Criteria{
			Terms:             []yacymodel.Hash{word1, word2},
			RequiredDocuments: []yacymodel.URLHash{searchtest.URLHashFor("u1")},
		},
		map[yacymodel.Hash]termpostings.Match{
			word1: matchOf("u1", "u2"),
			word2: matchOf("u2"),
		},
		nil,
	)
	if abstracts != nil {
		t.Fatalf("abstracts = %v, want none", abstracts)
	}
}

func TestIndexAbstractsForTermWithMostPostingsBreakTiesBySmallerTerm(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.IndexAbstractOfTermWithMostPostings{},
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word1, word2}},
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

func TestIndexAbstractsForTermWithMostPostingsStayEmptyWithoutMatches(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.IndexAbstractOfTermWithMostPostings{},
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word1, word2}},
		nil,
		nil,
	)
	if abstracts != nil {
		t.Fatalf("abstracts = %v, want none", abstracts)
	}
}

func TestIndexAbstractsOfTermsListTheDocumentsBehindThem(t *testing.T) {
	word, related := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	abstracts := indexabstract.IndexAbstractsFor(
		indexabstract.IndexAbstractsOfTerms{Terms: []yacymodel.Hash{related}},
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}},
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

func TestIndexAbstractTermsOfNamesOnlyTheTermsAnAbstractOfTermsCovers(t *testing.T) {
	related := searchtest.HashFor("w2")

	if len(indexabstract.IndexAbstractTermsOf(
		indexabstract.IndexAbstractsOfTerms{Terms: []yacymodel.Hash{related}},
	)) != 1 {
		t.Error("IndexAbstractTermsOf named no terms, want w2")
	}
	quiet := []indexabstract.RequestedIndexAbstracts{
		indexabstract.NoIndexAbstracts{},
		indexabstract.IndexAbstractOfTermWithMostPostings{},
	}
	for _, requested := range quiet {
		if terms := indexabstract.IndexAbstractTermsOf(requested); terms != nil {
			t.Errorf("IndexAbstractTermsOf(%T) = %v, want none", requested, terms)
		}
	}
}
