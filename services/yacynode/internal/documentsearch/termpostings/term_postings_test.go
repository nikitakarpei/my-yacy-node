package termpostings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

func postingEntry(url string, hits int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{URLHash: searchtest.URLHashFor(url), Hits: hits}
}

func matchesFor(
	t *testing.T,
	index rwipostings.PostingIndex,
	maxPostingsPerTerm int,
	terms []yacymodel.Hash,
	filter postingfilter.Filter,
) (map[yacymodel.Hash]termpostings.Match, error) {
	t.Helper()

	var matches map[yacymodel.Hash]termpostings.Match
	err := inReadTransaction(t, func(ctx context.Context, tx *vault.Txn) error {
		found, err := termpostings.New(index, maxPostingsPerTerm).
			MatchesFor(ctx, tx, terms, filter)
		matches = found

		return err
	})

	return matches, err
}

func inReadTransaction(
	t *testing.T,
	read func(ctx context.Context, tx *vault.Txn) error,
) error {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = v.Close() })

	ctx := context.Background()

	return v.View(ctx, func(tx *vault.Txn) error { return read(ctx, tx) })
}

func everyPosting() postingfilter.Filter {
	return postingfilter.FilterForReport(searchcriteria.Criteria{})
}

func TestMatchesForHoldsOnePostingPerDocument(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry("u1", 1), postingEntry("u2", 3)},
	}}

	matches, err := matchesFor(t, index, 100, []yacymodel.Hash{word}, everyPosting())
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	match := matches[word]
	if match.TotalMatches != 2 || len(match.PostingPerDocument) != 2 {
		t.Fatalf("match = %+v, want two postings", match)
	}
	if match.PostingPerDocument[searchtest.URLHashFor("u2")].Occurrences != 3 {
		t.Errorf("occurrences = %+v, want 3", match.PostingPerDocument[searchtest.URLHashFor("u2")])
	}
}

func TestMatchesForKeepsMostFrequentPostingsUnderCap(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry("u1", 1), postingEntry("u2", 7), postingEntry("u3", 4)},
	}}

	matches, err := matchesFor(t, index, 1, []yacymodel.Hash{word}, everyPosting())
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	match := matches[word]
	if match.TotalMatches != 3 {
		t.Errorf("TotalMatches = %d, want 3", match.TotalMatches)
	}
	if len(match.PostingPerDocument) != 1 {
		t.Fatalf("kept = %d postings, want 1", len(match.PostingPerDocument))
	}
	if _, ok := match.PostingPerDocument[searchtest.URLHashFor("u2")]; !ok {
		t.Error("most frequent posting not kept")
	}
}

func TestMatchesForKeepsTwoMostFrequentPostingsUnderCap(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {
			postingEntry("u1", 5),
			postingEntry("u2", 1),
			postingEntry("u3", 7),
			postingEntry("u4", 3),
		},
	}}

	matches, err := matchesFor(t, index, 2, []yacymodel.Hash{word}, everyPosting())
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	match := matches[word]
	if match.TotalMatches != 4 {
		t.Errorf("TotalMatches = %d, want 4", match.TotalMatches)
	}
	if len(match.PostingPerDocument) != 2 {
		t.Fatalf("kept = %d postings, want 2", len(match.PostingPerDocument))
	}
	for _, url := range []string{"u1", "u3"} {
		if _, ok := match.PostingPerDocument[searchtest.URLHashFor(url)]; !ok {
			t.Errorf("posting %s not kept, want the two most frequent", url)
		}
	}
}

func TestMatchesForSurfacesScanFailures(t *testing.T) {
	index := searchtest.FailingPostingIndex{Err: errScanBroken}

	_, err := matchesFor(t, index, 100, []yacymodel.Hash{searchtest.HashFor("w1")}, everyPosting())
	if !errors.Is(err, errScanBroken) {
		t.Fatalf("MatchesFor error = %v, want %v", err, errScanBroken)
	}
}

var errScanBroken = errors.New("scan broken")

func TestMatchesForSkipsPostingsTheFilterRejects(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry("u1", 1)},
	}}
	rejectingFilter := postingfilter.FilterForReport(searchcriteria.Criteria{
		RequiredDocuments: []yacymodel.URLHash{searchtest.URLHashFor("u9")},
	})

	matches, err := matchesFor(t, index, 100, []yacymodel.Hash{word}, rejectingFilter)
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	if matches[word].TotalMatches != 0 {
		t.Errorf("TotalMatches = %d, want 0", matches[word].TotalMatches)
	}
}

func TestDocumentsContainingNamesEveryDocumentHoldingATerm(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry("u1", 1), postingEntry("u2", 3)},
	}}

	documents, err := documentsContaining(t, index, []yacymodel.Hash{word})
	if err != nil {
		t.Fatalf("DocumentsContaining: %v", err)
	}
	if len(documents) != 2 {
		t.Fatalf("named %d documents, want 2", len(documents))
	}
	for _, url := range []string{"u1", "u2"} {
		if _, ok := documents[searchtest.URLHashFor(url)]; !ok {
			t.Errorf("document %s not named", url)
		}
	}
}

func documentsContaining(
	t *testing.T,
	index rwipostings.PostingIndex,
	terms []yacymodel.Hash,
) (map[yacymodel.URLHash]struct{}, error) {
	t.Helper()

	var documents map[yacymodel.URLHash]struct{}
	err := inReadTransaction(t, func(ctx context.Context, tx *vault.Txn) error {
		found, err := termpostings.New(index, 100).DocumentsContaining(ctx, tx, terms)
		documents = found

		return err
	})

	return documents, err
}

func TestDocumentsContainingSurfacesScanFailures(t *testing.T) {
	index := searchtest.FailingPostingIndex{Err: errScanBroken}

	_, err := documentsContaining(t, index, []yacymodel.Hash{searchtest.HashFor("w1")})
	if !errors.Is(err, errScanBroken) {
		t.Fatalf("DocumentsContaining error = %v, want %v", err, errScanBroken)
	}
}
