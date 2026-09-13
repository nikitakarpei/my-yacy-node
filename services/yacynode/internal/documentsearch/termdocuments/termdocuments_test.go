package termdocuments_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termdocuments"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiimpactorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

const indexAbstractDocumentsPerTerm = 1000

type termIndex interface {
	rwipostings.PostingIndex
	rwiimpactorder.ImpactOrderQuery
}

func postingOf(word yacymodel.Hash, document string, hits int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash: word,
		URLHash:  searchtest.URLHashFor(document),
		Hits:     hits,
	}
}

func documentsHoldingTerm(
	t *testing.T,
	index termIndex,
	term yacymodel.Hash,
	criteria searchcriteria.Criteria,
	indexAbstractDocumentsPerTerm int,
) ([]yacymodel.URLHash, error) {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = v.Close() })

	var documents []yacymodel.URLHash

	ctx := context.Background()
	err = v.View(ctx, func(tx *vault.Txn) error {
		found, readErr := termdocuments.New(
			index, index, indexAbstractDocumentsPerTerm,
		).DocumentsHoldingTerm(ctx, tx, term, criteria)
		documents = found

		return readErr
	})

	return documents, err
}

func TestDocumentsHoldingTermComeInImpactOrder(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingOf(word, "u1", 1), postingOf(word, "u2", 9)},
	}}

	documents, err := documentsHoldingTerm(
		t, index, word, searchcriteria.Criteria{}, indexAbstractDocumentsPerTerm,
	)
	if err != nil {
		t.Fatalf("DocumentsHoldingTerm: %v", err)
	}
	if len(documents) != 2 || documents[0] != searchtest.URLHashFor("u2") {
		t.Errorf("documents = %v, want the most relevant first", documents)
	}
}

func TestDocumentsHoldingTermStopAtTheDocumentsAnIndexAbstractCovers(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingOf(word, "u1", 1), postingOf(word, "u2", 9), postingOf(word, "u3", 5)},
	}}

	documents, err := documentsHoldingTerm(t, index, word, searchcriteria.Criteria{}, 2)
	if err != nil {
		t.Fatalf("DocumentsHoldingTerm: %v", err)
	}
	if len(documents) != 2 {
		t.Errorf("documents = %v, want two", documents)
	}
}

func TestDocumentsHoldingTermOutnumberTheResultsTheRequestAsksFor(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingOf(word, "u1", 1), postingOf(word, "u2", 9), postingOf(word, "u3", 5)},
	}}

	documents, err := documentsHoldingTerm(
		t,
		index,
		word,
		searchcriteria.Criteria{MaxResults: 1},
		indexAbstractDocumentsPerTerm,
	)
	if err != nil {
		t.Fatalf("DocumentsHoldingTerm: %v", err)
	}
	if len(documents) != 3 {
		t.Errorf("documents = %v, want every document of the term", documents)
	}
}

func TestDocumentsHoldingTermSkipDocumentsTheFilterRejects(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingOf(word, "u1", 1), postingOf(word, "u2", 9)},
	}}

	documents, err := documentsHoldingTerm(
		t,
		index,
		word,
		searchcriteria.Criteria{
			RequiredDocuments: []yacymodel.URLHash{searchtest.URLHashFor("u1")},
		},
		indexAbstractDocumentsPerTerm,
	)
	if err != nil {
		t.Fatalf("DocumentsHoldingTerm: %v", err)
	}
	if len(documents) != 1 || documents[0] != searchtest.URLHashFor("u1") {
		t.Errorf("documents = %v, want only the required document", documents)
	}
}

func TestDocumentsHoldingTermSurfaceIndexFailures(t *testing.T) {
	index := searchtest.FailingPostingIndex{Err: errIndexBroken}

	_, err := documentsHoldingTerm(
		t,
		index,
		searchtest.HashFor("w1"),
		searchcriteria.Criteria{},
		indexAbstractDocumentsPerTerm,
	)
	if !errors.Is(err, errIndexBroken) {
		t.Fatalf("error = %v, want %v", err, errIndexBroken)
	}
}

var errIndexBroken = errors.New("index broken")
