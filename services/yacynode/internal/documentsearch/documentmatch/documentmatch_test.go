package documentmatch_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/documentmatch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiimpactorder"
)

func postingOf(word yacymodel.Hash, document string, hits int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash: word,
		URLHash:  searchtest.URLHashFor(document),
		Hits:     hits,
	}
}

func titlePostingOf(word yacymodel.Hash, document string) yacymodel.RWIPosting {
	posting := postingOf(word, document, 1)
	posting.Appearance.AppearsInTitle = true

	return posting
}

type searchIndex interface {
	PostingOf(
		tx *vault.Txn,
		word yacymodel.Hash,
		document yacymodel.URLHash,
	) (yacymodel.RWIPosting, bool, error)
	RWICount(tx *vault.Txn) (int, error)
	ScanPostingsInImpactOrder(
		tx *vault.Txn,
		word yacymodel.Hash,
		visit func(yacymodel.URLHash, rwiimpactorder.Impact) (bool, error),
	) error
	LargestImpactOf(tx *vault.Txn, word yacymodel.Hash) (rwiimpactorder.Impact, bool, error)
	AmountOfPostingsOf(tx *vault.Txn, word yacymodel.Hash) (int, error)
}

func mostRelevantDocumentsFor(
	t *testing.T,
	index searchIndex,
	criteria searchcriteria.Criteria,
) (documentmatch.MostRelevantDocuments, error) {
	t.Helper()

	return mostRelevantDocumentsWithin(t, context.Background(), index, criteria)
}

func mostRelevantDocumentsWithin(
	t *testing.T,
	ctx context.Context,
	index searchIndex,
	criteria searchcriteria.Criteria,
) (documentmatch.MostRelevantDocuments, error) {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = v.Close() })

	amountOfPostingsPerTerm := make(map[yacymodel.Hash]int, len(criteria.Terms))

	var documents documentmatch.MostRelevantDocuments

	err = v.View(ctx, func(tx *vault.Txn) error {
		for _, term := range criteria.Terms {
			amount, amountErr := index.AmountOfPostingsOf(tx, term)
			if amountErr != nil {
				return amountErr
			}
			amountOfPostingsPerTerm[term] = amount
		}
		found, matchErr := documentmatch.New(index, index).MostRelevantDocumentsFor(
			ctx, tx, criteria, amountOfPostingsPerTerm,
		)
		documents = found

		return matchErr
	})

	return documents, err
}

func documentNames(documents documentmatch.MostRelevantDocuments) []string {
	names := make([]string, 0, len(documents.Postings))
	for _, posting := range documents.Postings {
		names = append(names, posting.URLHash.String())
	}

	return names
}

func TestMostRelevantDocumentsComeInImpactOrder(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingOf(word, "u1", 50), titlePostingOf(word, "u2")},
	}}

	documents, err := mostRelevantDocumentsFor(t, index, searchcriteria.Criteria{
		Terms:      []yacymodel.Hash{word},
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("MostRelevantDocumentsFor: %v", err)
	}
	if len(documents.Postings) != 2 {
		t.Fatalf("documents = %v, want two", documentNames(documents))
	}
	if documents.Postings[0].URLHash != searchtest.URLHashFor("u2") {
		t.Errorf(
			"documents = %v, want the title posting first",
			documentNames(documents),
		)
	}
}

func TestMostRelevantDocumentsStopOnceNoUnreadDocumentCanEnterTheAnswer(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {
			titlePostingOf(word, "u1"),
			postingOf(word, "u2", 9),
			postingOf(word, "u3", 5),
			postingOf(word, "u4", 3),
			postingOf(word, "u5", 1),
		},
	}}

	documents, err := mostRelevantDocumentsFor(t, index, searchcriteria.Criteria{
		Terms:      []yacymodel.Hash{word},
		MaxResults: 1,
	})
	if err != nil {
		t.Fatalf("MostRelevantDocumentsFor: %v", err)
	}
	if documents.IndexReadStop != documentmatch.StoppedAtRelevanceBound {
		t.Errorf("index read stop = %v, want the relevance bound", documents.IndexReadStop)
	}
	if len(documents.Postings) != 1 ||
		documents.Postings[0].URLHash != searchtest.URLHashFor("u1") {
		t.Errorf("documents = %v, want the title posting alone", documentNames(documents))
	}
}

func TestMostRelevantDocumentsNeverReadACommonWordInFull(t *testing.T) {
	rareWord, commonWord := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	postings := map[yacymodel.Hash][]yacymodel.RWIPosting{
		rareWord:   {postingOf(rareWord, "u1", 1)},
		commonWord: {postingOf(commonWord, "u1", 1)},
	}
	for _, document := range []string{"u2", "u3", "u4", "u5", "u6"} {
		postings[commonWord] = append(postings[commonWord], postingOf(commonWord, document, 1))
	}
	index := &countingIndex{PostingIndex: searchtest.PostingIndex{Postings: postings}}

	documents, err := mostRelevantDocumentsFor(t, index, searchcriteria.Criteria{
		Terms:      []yacymodel.Hash{commonWord, rareWord},
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("MostRelevantDocumentsFor: %v", err)
	}
	if len(documents.Postings) != 1 {
		t.Fatalf("documents = %v, want the one document both words hold", documentNames(documents))
	}
	if index.readDocumentsOf[commonWord] != 0 {
		t.Errorf(
			"read %d postings of the common word in order, want none",
			index.readDocumentsOf[commonWord],
		)
	}
}

type countingIndex struct {
	searchtest.PostingIndex
	readDocumentsOf map[yacymodel.Hash]int
}

func (index *countingIndex) ScanPostingsInImpactOrder(
	tx *vault.Txn,
	word yacymodel.Hash,
	visit func(yacymodel.URLHash, rwiimpactorder.Impact) (bool, error),
) error {
	if index.readDocumentsOf == nil {
		index.readDocumentsOf = map[yacymodel.Hash]int{}
	}

	return index.PostingIndex.ScanPostingsInImpactOrder(
		tx,
		word,
		func(document yacymodel.URLHash, impact rwiimpactorder.Impact) (bool, error) {
			index.readDocumentsOf[word]++

			return visit(document, impact)
		},
	)
}

func TestMostRelevantDocumentsDropDocumentsHoldingAnExcludedTerm(t *testing.T) {
	word, excluded := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word:     {postingOf(word, "u1", 1), postingOf(word, "u2", 1)},
		excluded: {postingOf(excluded, "u2", 1)},
	}}

	documents, err := mostRelevantDocumentsFor(t, index, searchcriteria.Criteria{
		Terms:         []yacymodel.Hash{word},
		ExcludedTerms: []yacymodel.Hash{excluded},
		MaxResults:    10,
	})
	if err != nil {
		t.Fatalf("MostRelevantDocumentsFor: %v", err)
	}
	if len(documents.Postings) != 1 ||
		documents.Postings[0].URLHash != searchtest.URLHashFor("u1") {
		t.Errorf("documents = %v, want only u1", documentNames(documents))
	}
}

func TestMostRelevantDocumentsDropDocumentsTheFilterRejects(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingOf(word, "u1", 1), postingOf(word, "u2", 1)},
	}}

	documents, err := mostRelevantDocumentsFor(t, index, searchcriteria.Criteria{
		Terms:             []yacymodel.Hash{word},
		RequiredDocuments: []yacymodel.URLHash{searchtest.URLHashFor("u2")},
		MaxResults:        10,
	})
	if err != nil {
		t.Fatalf("MostRelevantDocumentsFor: %v", err)
	}
	if len(documents.Postings) != 1 ||
		documents.Postings[0].URLHash != searchtest.URLHashFor("u2") {
		t.Errorf("documents = %v, want only the required document", documentNames(documents))
	}
}

func TestMostRelevantDocumentsStayEmptyWhenTheNodeHoldsNoPostingOfATerm(t *testing.T) {
	word, absent := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingOf(word, "u1", 1)},
	}}

	documents, err := mostRelevantDocumentsFor(t, index, searchcriteria.Criteria{
		Terms:      []yacymodel.Hash{word, absent},
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("MostRelevantDocumentsFor: %v", err)
	}
	if len(documents.Postings) != 0 || documents.AmountOfMatchedDocuments != 0 {
		t.Errorf("documents = %v, want none", documentNames(documents))
	}
}

func TestMostRelevantDocumentsAnswerWithWhatTheyFoundWhenTheRequestEnds(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {
			postingOf(word, "u1", 9),
			postingOf(word, "u2", 5),
			postingOf(word, "u3", 1),
		},
	}}

	ctx, endRequest := context.WithCancel(context.Background())
	defer endRequest()

	documents, err := mostRelevantDocumentsWithin(
		t,
		ctx,
		&endingIndex{
			PostingIndex: index,
			endRequest:   endRequest,
		},
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}, MaxResults: 10},
	)
	if err != nil {
		t.Fatalf("MostRelevantDocumentsFor: %v", err)
	}
	if documents.IndexReadStop != documentmatch.StoppedAtDeadline {
		t.Errorf("index read stop = %v, want the deadline", documents.IndexReadStop)
	}
	if len(documents.Postings) != 1 {
		t.Errorf("documents = %v, want the one document read before the end",
			documentNames(documents))
	}
}

type endingIndex struct {
	searchtest.PostingIndex
	endRequest context.CancelFunc
}

func (index *endingIndex) ScanPostingsInImpactOrder(
	tx *vault.Txn,
	word yacymodel.Hash,
	visit func(yacymodel.URLHash, rwiimpactorder.Impact) (bool, error),
) error {
	return index.PostingIndex.ScanPostingsInImpactOrder(
		tx,
		word,
		func(document yacymodel.URLHash, impact rwiimpactorder.Impact) (bool, error) {
			keepGoing, err := visit(document, impact)
			index.endRequest()

			return keepGoing, err
		},
	)
}

func TestMostRelevantDocumentsSurfaceIndexFailures(t *testing.T) {
	word := searchtest.HashFor("w1")

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = v.Close() })

	index := searchtest.FailingPostingIndex{Err: errIndexBroken}
	ctx := context.Background()
	err = v.View(ctx, func(tx *vault.Txn) error {
		_, matchErr := documentmatch.New(index, index).MostRelevantDocumentsFor(
			ctx,
			tx,
			searchcriteria.Criteria{Terms: []yacymodel.Hash{word}, MaxResults: 10},
			map[yacymodel.Hash]int{word: 1},
		)

		return matchErr
	})
	if !errors.Is(err, errIndexBroken) {
		t.Fatalf("error = %v, want %v", err, errIndexBroken)
	}
}

var errIndexBroken = errors.New("index broken")
