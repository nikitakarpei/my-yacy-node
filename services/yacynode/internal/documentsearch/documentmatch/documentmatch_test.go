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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
)

func postingOf(term yacymodel.Hash, document string, hits int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash: term,
		URLHash:  searchtest.URLHashFor(document),
		Hits:     hits,
	}
}

func titlePostingOf(term yacymodel.Hash, document string) yacymodel.RWIPosting {
	posting := postingOf(term, document, 1)
	posting.Appearance.AppearsInTitle = true

	return posting
}

type searchIndex interface {
	PostingOf(
		tx *vault.Txn,
		term yacymodel.Hash,
		document yacymodel.URLHash,
	) (yacymodel.RWIPosting, bool, error)
	RWICount(tx *vault.Txn) (int, error)
	ScanPostingsInImpactOrder(
		tx *vault.Txn,
		term yacymodel.Hash,
		visit func(yacymodel.URLHash, rwipostingimpactorder.Impact) (bool, error),
	) error
	LargestImpactOf(tx *vault.Txn, term yacymodel.Hash) (rwipostingimpactorder.Impact, bool, error)
	AmountOfPostingsOf(tx *vault.Txn, term yacymodel.Hash) (int, error)
}

func matchesFor(
	t *testing.T,
	index searchIndex,
	criteria searchcriteria.Criteria,
) (documentmatch.DocumentMatches, error) {
	t.Helper()

	return matchesWithin(t, context.Background(), index, criteria)
}

func matchesWithin(
	t *testing.T,
	ctx context.Context,
	index searchIndex,
	criteria searchcriteria.Criteria,
) (documentmatch.DocumentMatches, error) {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = v.Close() })

	amountOfPostingsPerTerm := make(map[yacymodel.Hash]int, len(criteria.Terms))

	var matches documentmatch.DocumentMatches

	err = v.View(ctx, func(tx *vault.Txn) error {
		for _, term := range criteria.Terms {
			amount, amountErr := index.AmountOfPostingsOf(tx, term)
			if amountErr != nil {
				return amountErr
			}
			amountOfPostingsPerTerm[term] = amount
		}
		found, matchErr := documentmatch.New(index, index).MatchesFor(
			ctx, tx, criteria, amountOfPostingsPerTerm,
		)
		matches = found

		return matchErr
	})

	return matches, err
}

func documentNames(matches documentmatch.DocumentMatches) []string {
	names := make([]string, 0, len(matches.JoinedPostings))
	for _, posting := range matches.JoinedPostings {
		names = append(names, posting.URLHash.String())
	}

	return names
}

func TestMostRelevantDocumentsComeInImpactOrder(t *testing.T) {
	term := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		term: {postingOf(term, "u1", 50), titlePostingOf(term, "u2")},
	}}

	matches, err := matchesFor(t, index, searchcriteria.Criteria{
		Terms:      []yacymodel.Hash{term},
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	if len(matches.JoinedPostings) != 2 {
		t.Fatalf("documents = %v, want two", documentNames(matches))
	}
	if matches.JoinedPostings[0].URLHash != searchtest.URLHashFor("u2") {
		t.Errorf(
			"documents = %v, want the title posting first",
			documentNames(matches),
		)
	}
}

func TestMostRelevantDocumentsStopOnceTheyReachTheRelevanceBound(t *testing.T) {
	term := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		term: {
			titlePostingOf(term, "u1"),
			postingOf(term, "u2", 9),
			postingOf(term, "u3", 5),
			postingOf(term, "u4", 3),
			postingOf(term, "u5", 1),
		},
	}}

	matches, err := matchesFor(t, index, searchcriteria.Criteria{
		Terms:      []yacymodel.Hash{term},
		MaxResults: 1,
	})
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	if matches.IndexReadStopReason != documentmatch.IndexReadStoppedAtRelevanceBound {
		t.Errorf(
			"index read stop = %v, want the relevance bound",
			matches.IndexReadStopReason,
		)
	}
	if len(matches.JoinedPostings) != 1 ||
		matches.JoinedPostings[0].URLHash != searchtest.URLHashFor("u1") {
		t.Errorf(
			"documents = %v, want the title posting alone",
			documentNames(matches),
		)
	}
}

func TestMostRelevantDocumentsNeverReadACommonTermInFull(t *testing.T) {
	rareTerm, commonTerm := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	postings := map[yacymodel.Hash][]yacymodel.RWIPosting{
		rareTerm:   {postingOf(rareTerm, "u1", 1)},
		commonTerm: {postingOf(commonTerm, "u1", 1)},
	}
	for _, document := range []string{"u2", "u3", "u4", "u5", "u6"} {
		postings[commonTerm] = append(postings[commonTerm], postingOf(commonTerm, document, 1))
	}
	index := &countingIndex{PostingIndex: searchtest.PostingIndex{Postings: postings}}

	matches, err := matchesFor(t, index, searchcriteria.Criteria{
		Terms:      []yacymodel.Hash{commonTerm, rareTerm},
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	if len(matches.JoinedPostings) != 1 {
		t.Fatalf(
			"documents = %v, want the one document both terms hold",
			documentNames(matches),
		)
	}
	if index.readDocumentsOf[commonTerm] != 0 {
		t.Errorf(
			"read %d postings of the common term in order, want none",
			index.readDocumentsOf[commonTerm],
		)
	}
}

type countingIndex struct {
	searchtest.PostingIndex
	readDocumentsOf map[yacymodel.Hash]int
}

func (index *countingIndex) ScanPostingsInImpactOrder(
	tx *vault.Txn,
	term yacymodel.Hash,
	visit func(yacymodel.URLHash, rwipostingimpactorder.Impact) (bool, error),
) error {
	if index.readDocumentsOf == nil {
		index.readDocumentsOf = map[yacymodel.Hash]int{}
	}

	return index.PostingIndex.ScanPostingsInImpactOrder(
		tx,
		term,
		func(document yacymodel.URLHash, impact rwipostingimpactorder.Impact) (bool, error) {
			index.readDocumentsOf[term]++

			return visit(document, impact)
		},
	)
}

func TestMostRelevantDocumentsDropDocumentsHoldingAnExcludedTerm(t *testing.T) {
	term, excluded := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		term:     {postingOf(term, "u1", 1), postingOf(term, "u2", 1)},
		excluded: {postingOf(excluded, "u2", 1)},
	}}

	matches, err := matchesFor(t, index, searchcriteria.Criteria{
		Terms:         []yacymodel.Hash{term},
		ExcludedTerms: []yacymodel.Hash{excluded},
		MaxResults:    10,
	})
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	if len(matches.JoinedPostings) != 1 ||
		matches.JoinedPostings[0].URLHash != searchtest.URLHashFor("u1") {
		t.Errorf("documents = %v, want only u1", documentNames(matches))
	}
}

func TestMostRelevantDocumentsDropDocumentsTheFilterRejects(t *testing.T) {
	term := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		term: {postingOf(term, "u1", 1), postingOf(term, "u2", 1)},
	}}

	matches, err := matchesFor(t, index, searchcriteria.Criteria{
		Terms:             []yacymodel.Hash{term},
		RequiredDocuments: []yacymodel.URLHash{searchtest.URLHashFor("u2")},
		MaxResults:        10,
	})
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	if len(matches.JoinedPostings) != 1 ||
		matches.JoinedPostings[0].URLHash != searchtest.URLHashFor("u2") {
		t.Errorf(
			"documents = %v, want only the required document",
			documentNames(matches),
		)
	}
}

func TestMostRelevantDocumentsStayEmptyWhenTheNodeHoldsNoPostingOfATerm(t *testing.T) {
	term, absent := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		term: {postingOf(term, "u1", 1)},
	}}

	matches, err := matchesFor(t, index, searchcriteria.Criteria{
		Terms:      []yacymodel.Hash{term, absent},
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	if len(matches.JoinedPostings) != 0 ||
		matches.AmountOfDocumentsMatchingEveryTerm != 0 {
		t.Errorf("documents = %v, want none", documentNames(matches))
	}
}

func TestMostRelevantDocumentsStayEmptyWhenTheSearchNamesNoTerm(t *testing.T) {
	term := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		term: {postingOf(term, "u1", 1)},
	}}

	matches, err := matchesFor(t, index, searchcriteria.Criteria{MaxResults: 10})
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	if len(matches.JoinedPostings) != 0 ||
		matches.AmountOfDocumentsMatchingEveryTerm != 0 {
		t.Errorf("documents = %v, want none", documentNames(matches))
	}
}

func TestMostRelevantDocumentsAnswerWithWhatTheyFoundWhenTheRequestEnds(t *testing.T) {
	term := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		term: {
			postingOf(term, "u1", 9),
			postingOf(term, "u2", 5),
			postingOf(term, "u3", 1),
		},
	}}

	ctx, endRequest := context.WithCancel(context.Background())
	defer endRequest()

	matches, err := matchesWithin(
		t,
		ctx,
		&endingIndex{
			PostingIndex: index,
			endRequest:   endRequest,
		},
		searchcriteria.Criteria{Terms: []yacymodel.Hash{term}, MaxResults: 10},
	)
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	if matches.IndexReadStopReason != documentmatch.IndexReadStoppedAtDeadline {
		t.Errorf("index read stop = %v, want the deadline", matches.IndexReadStopReason)
	}
	if len(matches.JoinedPostings) != 1 {
		t.Errorf("documents = %v, want the one document read before the end",
			documentNames(matches))
	}
}

type endingIndex struct {
	searchtest.PostingIndex
	endRequest context.CancelFunc
}

func (index *endingIndex) ScanPostingsInImpactOrder(
	tx *vault.Txn,
	term yacymodel.Hash,
	visit func(yacymodel.URLHash, rwipostingimpactorder.Impact) (bool, error),
) error {
	return index.PostingIndex.ScanPostingsInImpactOrder(
		tx,
		term,
		func(document yacymodel.URLHash, impact rwipostingimpactorder.Impact) (bool, error) {
			keepGoing, err := visit(document, impact)
			index.endRequest()

			return keepGoing, err
		},
	)
}

func TestMostRelevantDocumentsSurfaceIndexFailures(t *testing.T) {
	term := searchtest.HashFor("w1")

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = v.Close() })

	index := searchtest.FailingPostingIndex{Err: errIndexBroken}
	ctx := context.Background()
	err = v.View(ctx, func(tx *vault.Txn) error {
		_, matchErr := documentmatch.New(index, index).MatchesFor(
			ctx,
			tx,
			searchcriteria.Criteria{Terms: []yacymodel.Hash{term}, MaxResults: 10},
			map[yacymodel.Hash]int{term: 1},
		)

		return matchErr
	})
	if !errors.Is(err, errIndexBroken) {
		t.Fatalf("error = %v, want %v", err, errIndexBroken)
	}
}

var errIndexBroken = errors.New("index broken")
