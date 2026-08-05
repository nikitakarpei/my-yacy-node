package termmatch

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
)

type fakeScanner struct {
	postings map[yacymodel.Hash][]yacymodel.RWIPosting
}

func (s fakeScanner) RWICount(context.Context) (int, error) {
	return len(s.postings), nil
}

func (s fakeScanner) ScanWord(
	_ context.Context,
	word yacymodel.Hash,
	visit func(yacymodel.RWIPosting) (bool, error),
) error {
	for _, entry := range s.postings[word] {
		entry.WordHash = word
		keepGoing, err := visit(entry)
		if err != nil {
			return err
		}
		if !keepGoing {
			return nil
		}
	}

	return nil
}

func (s fakeScanner) PostingOf(
	_ context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	for _, entry := range s.postings[word] {
		if entry.URLHash == url {
			entry.WordHash = word

			return entry, true, nil
		}
	}

	return yacymodel.RWIPosting{}, false, nil
}

func hashFor(base string) yacymodel.Hash {
	const filler = "AAAAAAAAAAAA"
	hash, err := yacymodel.ParseHash(base + filler[len(base):])
	if err != nil {
		panic(err)
	}

	return hash
}

func urlHashFor(url string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(hashFor(url).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func postingEntry(url string, hits int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{URLHash: urlHashFor(url), Hits: hits}
}

func TestMatchesForHoldsOnePostingPerDocument(t *testing.T) {
	word := hashFor("w1")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry("u1", 1), postingEntry("u2", 3)},
	}}

	matches, err := MatchesFor(
		context.Background(),
		index,
		[]yacymodel.Hash{word},
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
	if match.PostingPerDocument[urlHashFor("u2")].Occurrences != 3 {
		t.Errorf("occurrences = %+v, want 3", match.PostingPerDocument[urlHashFor("u2")])
	}
}

func TestMatchesForKeepsMostFrequentPostingsUnderCap(t *testing.T) {
	word := hashFor("w1")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry("u1", 1), postingEntry("u2", 7), postingEntry("u3", 4)},
	}}

	matches, err := MatchesFor(
		context.Background(),
		index,
		[]yacymodel.Hash{word},
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
	if _, ok := match.PostingPerDocument[urlHashFor("u2")]; !ok {
		t.Error("most frequent posting not kept")
	}
}

func TestMatchesForSkipsPostingsTheFilterRejects(t *testing.T) {
	word := hashFor("w1")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry("u1", 1)},
	}}

	matches, err := MatchesFor(
		context.Background(),
		index,
		[]yacymodel.Hash{word},
		postingfilter.FilterForReport(searchcriteria.Criteria{
			RequiredDocuments: []yacymodel.URLHash{urlHashFor("u9")},
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
