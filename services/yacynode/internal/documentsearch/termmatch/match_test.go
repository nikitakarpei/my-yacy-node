package termmatch_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termmatch"
)

func postingEntry(url string, hits int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{URLHash: searchtest.URLHashFor(url), Hits: hits}
}

func TestMatchesForHoldsOnePostingPerDocument(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry("u1", 1), postingEntry("u2", 3)},
	}}

	matches, err := termmatch.MatchesFor(
		context.Background(),
		[]yacymodel.Hash{word},
		index,
		postingfilter.FilterForReport(searchcriteria.Criteria{}),
		100,
	)
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

	matches, err := termmatch.MatchesFor(
		context.Background(),
		[]yacymodel.Hash{word},
		index,
		postingfilter.FilterForReport(searchcriteria.Criteria{}),
		1,
	)
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

	matches, err := termmatch.MatchesFor(
		context.Background(),
		[]yacymodel.Hash{word},
		index,
		postingfilter.FilterForReport(searchcriteria.Criteria{}),
		2,
	)
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
	_, err := termmatch.MatchesFor(
		context.Background(),
		[]yacymodel.Hash{searchtest.HashFor("w1")},
		searchtest.FailingPostingIndex{Err: errScanBroken},
		postingfilter.FilterForReport(searchcriteria.Criteria{}),
		100,
	)
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

	matches, err := termmatch.MatchesFor(
		context.Background(),
		[]yacymodel.Hash{word},
		index,
		postingfilter.FilterForReport(searchcriteria.Criteria{
			RequiredDocuments: []yacymodel.URLHash{searchtest.URLHashFor("u9")},
		}),
		100,
	)
	if err != nil {
		t.Fatalf("MatchesFor: %v", err)
	}
	if matches[word].TotalMatches != 0 {
		t.Errorf("TotalMatches = %d, want 0", matches[word].TotalMatches)
	}
}
