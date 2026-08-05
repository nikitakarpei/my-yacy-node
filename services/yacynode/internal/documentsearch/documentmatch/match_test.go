package documentmatch

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termmatch"
)

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

func matchOf(postings ...termmatch.Posting) termmatch.Match {
	byDocument := make(map[yacymodel.URLHash]termmatch.Posting, len(postings))
	for _, posting := range postings {
		byDocument[posting.DocumentHash] = posting
	}

	return termmatch.Match{PostingPerDocument: byDocument, TotalMatches: len(postings)}
}

func postingIn(url string, occurrences, textPosition int) termmatch.Posting {
	return termmatch.Posting{
		DocumentHash: urlHashFor(url),
		Occurrences:  occurrences,
		TextPosition: textPosition,
	}
}

func TestMatchesAcrossEveryTermSumsOccurrencesOfSharedDocuments(t *testing.T) {
	word1, word2 := hashFor("w1"), hashFor("w2")
	matches := MatchesAcrossEveryTerm(
		[]yacymodel.Hash{word1, word2},
		map[yacymodel.Hash]termmatch.Match{
			word1: matchOf(postingIn("u1", 1, 1), postingIn("u2", 2, 4)),
			word2: matchOf(postingIn("u2", 3, 8)),
		},
	)

	if len(matches) != 1 {
		t.Fatalf("matches = %d, want only the shared document", len(matches))
	}
	shared := matches[urlHashFor("u2")]
	if shared.termOccurrences != 5 {
		t.Errorf("termOccurrences = %d, want 5", shared.termOccurrences)
	}
	if shared.firstTermPosition != 4 || shared.lastTermPosition != 8 {
		t.Errorf(
			"positions = %d..%d, want 4..8",
			shared.firstTermPosition,
			shared.lastTermPosition,
		)
	}
}

func TestMatchesAcrossEveryTermWithoutTerms(t *testing.T) {
	if matches := MatchesAcrossEveryTerm(nil, nil); matches != nil {
		t.Fatalf("matches = %v, want nil without terms", matches)
	}
}

func TestMatchesWithinTermSpreadDropsWidelySpreadDocuments(t *testing.T) {
	near, far := urlHashFor("u1"), urlHashFor("u2")
	matches := map[yacymodel.URLHash]Match{
		near: {documentHash: near, firstTermPosition: 1, lastTermPosition: 3},
		far:  {documentHash: far, firstTermPosition: 1, lastTermPosition: 21},
	}

	within := MatchesWithinTermSpread(matches, 5, 2)

	if len(within) != 1 {
		t.Fatalf("within = %d, want only the near document", len(within))
	}
	if _, ok := within[near]; !ok {
		t.Error("near document missing")
	}
}

func TestMatchesWithinTermSpreadKeepsEveryDocumentWithoutLimit(t *testing.T) {
	documentHash := urlHashFor("u1")
	matches := map[yacymodel.URLHash]Match{documentHash: {documentHash: documentHash}}

	if within := MatchesWithinTermSpread(matches, 0, 1); len(within) != 1 {
		t.Fatalf("within = %d, want every document", len(within))
	}
}

func TestHashesOfMostRelevantDocumentsOrdersByOccurrencesThenSpread(t *testing.T) {
	frequent, near, far := urlHashFor("u1"), urlHashFor("u2"), urlHashFor("u3")
	matches := map[yacymodel.URLHash]Match{
		frequent: {documentHash: frequent, termOccurrences: 9, lastTermPosition: 20},
		near:     {documentHash: near, termOccurrences: 4, lastTermPosition: 2},
		far:      {documentHash: far, termOccurrences: 4, lastTermPosition: 10},
	}

	ordered := HashesOfMostRelevantDocuments(matches, 2, 0)

	want := []yacymodel.URLHash{frequent, near, far}
	for position, documentHash := range want {
		if ordered[position] != documentHash {
			t.Fatalf("order = %v, want %v", ordered, want)
		}
	}
}

func TestHashesOfMostRelevantDocumentsBreaksTiesOnDocumentHash(t *testing.T) {
	first, second := urlHashFor("u1"), urlHashFor("u2")
	matches := map[yacymodel.URLHash]Match{
		first:  {documentHash: first},
		second: {documentHash: second},
	}

	ordered := HashesOfMostRelevantDocuments(matches, 1, 1)

	if len(ordered) != 1 || ordered[0] != first {
		t.Fatalf("order = %v, want only %v", ordered, first)
	}
}
