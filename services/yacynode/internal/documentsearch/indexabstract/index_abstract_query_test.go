package indexabstract_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/indexabstract"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

const documentsPerAnswer = 10000

type termIndex interface {
	rwipostings.PostingIndex
	rwipostingimpactorder.ImpactOrderQuery
}

func TestIndexAbstractsNameTheMostRelevantDocumentFirst(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingOf(word, "u1", 1), postingOf(word, "u2", 9)},
	}}

	documents, err := documentsInIndexAbstractOfTerm(
		t, index, word, searchcriteria.Criteria{}, documentsPerAnswer,
	)
	if err != nil {
		t.Fatalf("IndexAbstractsFor: %v", err)
	}
	if len(documents) != 2 || documents[0] != searchtest.URLHashFor("u2") {
		t.Errorf("documents = %v, want the most relevant first", documents)
	}
}

func TestIndexAbstractOfASingleTermListsUpToTheDocumentsPerAnswer(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingOf(word, "u1", 1), postingOf(word, "u2", 9), postingOf(word, "u3", 5)},
	}}

	documents, err := documentsInIndexAbstractOfTerm(t, index, word, searchcriteria.Criteria{}, 2)
	if err != nil {
		t.Fatalf("IndexAbstractsFor: %v", err)
	}
	if len(documents) != 2 {
		t.Errorf("documents = %v, want two", documents)
	}
}

func TestIndexAbstractsOfSeveralTermsShareTheDocumentsPerAnswer(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {postingOf(word1, "u1", 1), postingOf(word1, "u2", 9), postingOf(word1, "u3", 5)},
		word2: {postingOf(word2, "u1", 1), postingOf(word2, "u2", 9), postingOf(word2, "u3", 5)},
	}}

	abstracts, err := indexAbstractsFrom(
		t,
		indexabstract.New(index, index, 5),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word1, word2}},
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractsOfTerms{Terms: []yacymodel.Hash{word1, word2}},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("IndexAbstractsFor: %v", err)
	}
	if len(abstracts[word1]) != 2 || len(abstracts[word2]) != 2 {
		t.Errorf("abstracts = %v, want two documents per term", abstracts)
	}
}

func TestIndexAbstractListsEveryDocumentOfATermWithinItsShare(t *testing.T) {
	word := searchtest.HashFor("w1")
	postings := make([]yacymodel.RWIPosting, 0, 1500)
	for document := range cap(postings) {
		postings = append(postings, postingOf(word, fmt.Sprintf("u%d", document), 1))
	}
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: postings,
	}}

	documents, err := documentsInIndexAbstractOfTerm(
		t, index, word, searchcriteria.Criteria{}, documentsPerAnswer,
	)
	if err != nil {
		t.Fatalf("IndexAbstractsFor: %v", err)
	}
	if len(documents) != len(postings) {
		t.Errorf("documents = %d, want all %d", len(documents), len(postings))
	}
}

func TestIndexAbstractsOutnumberTheResultsTheRequestAsksFor(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingOf(word, "u1", 1), postingOf(word, "u2", 9), postingOf(word, "u3", 5)},
	}}

	documents, err := documentsInIndexAbstractOfTerm(
		t,
		index,
		word,
		searchcriteria.Criteria{MaxResults: 1},
		documentsPerAnswer,
	)
	if err != nil {
		t.Fatalf("IndexAbstractsFor: %v", err)
	}
	if len(documents) != 3 {
		t.Errorf("documents = %v, want every document of the term", documents)
	}
}

func TestIndexAbstractsSkipDocumentsTheCriteriaReject(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingOf(word, "u1", 1), postingOf(word, "u2", 9)},
	}}

	documents, err := documentsInIndexAbstractOfTerm(
		t,
		index,
		word,
		searchcriteria.Criteria{
			RequiredDocuments: []yacymodel.URLHash{searchtest.URLHashFor("u1")},
		},
		documentsPerAnswer,
	)
	if err != nil {
		t.Fatalf("IndexAbstractsFor: %v", err)
	}
	if len(documents) != 1 || documents[0] != searchtest.URLHashFor("u1") {
		t.Errorf("documents = %v, want only the required document", documents)
	}
}

func TestIndexAbstractsSurfaceIndexFailures(t *testing.T) {
	index := searchtest.FailingPostingIndex{Err: errIndexBroken}

	_, err := documentsInIndexAbstractOfTerm(
		t,
		index,
		searchtest.HashFor("w1"),
		searchcriteria.Criteria{},
		documentsPerAnswer,
	)
	if !errors.Is(err, errIndexBroken) {
		t.Fatalf("error = %v, want %v", err, errIndexBroken)
	}
}

var errIndexBroken = errors.New("index broken")

func postingOf(word yacymodel.Hash, document string, hits int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash: word,
		URLHash:  searchtest.URLHashFor(document),
		Hits:     hits,
	}
}

func documentsInIndexAbstractOfTerm(
	t *testing.T,
	index termIndex,
	term yacymodel.Hash,
	criteria searchcriteria.Criteria,
	documentsPerAnswer indexabstract.DocumentsPerAnswer,
) ([]yacymodel.URLHash, error) {
	t.Helper()

	criteria.Terms = []yacymodel.Hash{term}
	abstracts, err := indexAbstractsFrom(
		t,
		indexabstract.New(index, index, documentsPerAnswer),
		criteria,
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractsOfTerms{Terms: []yacymodel.Hash{term}},
		},
		nil,
	)

	return abstracts[term], err
}

func indexAbstractsFrom(
	t *testing.T,
	query indexabstract.IndexAbstractQuery,
	criteria searchcriteria.Criteria,
	requested indexabstract.RequestedIndexAbstracts,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) (indexabstract.IndexAbstracts, error) {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = v.Close() })

	var abstracts indexabstract.IndexAbstracts

	ctx := context.Background()
	err = v.View(ctx, func(tx *vault.Txn) error {
		found, readErr := query.IndexAbstractsFor(
			ctx, tx, criteria, requested, amountOfPostingsPerTerm,
		)
		abstracts = found

		return readErr
	})

	return abstracts, err
}
