package documentmatch_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/documentmatch"
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

func matchesOfTwoTerms(
	first, second termmatch.Match,
) map[yacymodel.URLHash]documentmatch.Match {
	firstTerm, secondTerm := hashFor("w1"), hashFor("w2")

	return documentmatch.MatchesAcrossEveryTerm(
		[]yacymodel.Hash{firstTerm, secondTerm},
		map[yacymodel.Hash]termmatch.Match{firstTerm: first, secondTerm: second},
	)
}

func assertOrder(t *testing.T, ordered, want []yacymodel.URLHash) {
	t.Helper()

	if len(ordered) != len(want) {
		t.Fatalf("order = %v, want %v", ordered, want)
	}
	for position, documentHash := range want {
		if ordered[position] != documentHash {
			t.Fatalf("order = %v, want %v", ordered, want)
		}
	}
}

func TestMatchesAcrossEveryTermDropsDocumentsMissingATerm(t *testing.T) {
	matches := matchesOfTwoTerms(
		matchOf(postingIn("u1", 1, 1), postingIn("u2", 2, 4)),
		matchOf(postingIn("u2", 3, 8)),
	)

	assertOrder(
		t,
		documentmatch.HashesOfMostRelevantDocuments(matches, 2, 0),
		[]yacymodel.URLHash{urlHashFor("u2")},
	)
}

func TestMatchesAcrossEveryTermSumsOccurrencesRatherThanTakingTheHighest(t *testing.T) {
	matches := matchesOfTwoTerms(
		matchOf(postingIn("u1", 4, 0), postingIn("u2", 5, 0)),
		matchOf(postingIn("u1", 4, 10), postingIn("u2", 2, 10)),
	)

	assertOrder(
		t,
		documentmatch.HashesOfMostRelevantDocuments(matches, 2, 0),
		[]yacymodel.URLHash{urlHashFor("u1"), urlHashFor("u2")},
	)
}

func TestMatchesAcrossEveryTermWithoutTerms(t *testing.T) {
	matches := documentmatch.MatchesAcrossEveryTerm(nil, nil)

	if len(matches) != 0 {
		t.Fatalf("matches = %v, want none without terms", matches)
	}
}

func TestMatchesWithinTermSpreadDropsWidelySpreadDocuments(t *testing.T) {
	matches := matchesOfTwoTerms(
		matchOf(postingIn("u1", 1, 1), postingIn("u2", 1, 1)),
		matchOf(postingIn("u1", 1, 3), postingIn("u2", 1, 21)),
	)

	within := documentmatch.MatchesWithinTermSpread(matches, 5, 2)

	assertOrder(
		t,
		documentmatch.HashesOfMostRelevantDocuments(within, 2, 0),
		[]yacymodel.URLHash{urlHashFor("u1")},
	)
}

func TestMatchesWithinTermSpreadKeepsEveryDocumentWithoutLimit(t *testing.T) {
	matches := matchesOfTwoTerms(
		matchOf(postingIn("u1", 1, 1), postingIn("u2", 1, 1)),
		matchOf(postingIn("u1", 1, 3), postingIn("u2", 1, 21)),
	)

	within := documentmatch.MatchesWithinTermSpread(matches, 0, 2)

	assertOrder(
		t,
		documentmatch.HashesOfMostRelevantDocuments(within, 2, 0),
		[]yacymodel.URLHash{urlHashFor("u1"), urlHashFor("u2")},
	)
}

func TestHashesOfMostRelevantDocumentsOrdersByOccurrencesThenSpread(t *testing.T) {
	matches := matchesOfTwoTerms(
		matchOf(postingIn("u1", 9, 0), postingIn("u2", 2, 0), postingIn("u3", 2, 0)),
		matchOf(postingIn("u1", 1, 20), postingIn("u2", 1, 2), postingIn("u3", 1, 10)),
	)

	assertOrder(
		t,
		documentmatch.HashesOfMostRelevantDocuments(matches, 2, 0),
		[]yacymodel.URLHash{urlHashFor("u1"), urlHashFor("u2"), urlHashFor("u3")},
	)
}

func TestHashesOfMostRelevantDocumentsBreaksTiesOnDocumentHash(t *testing.T) {
	matches := matchesOfTwoTerms(
		matchOf(postingIn("u1", 1, 0), postingIn("u2", 1, 0)),
		matchOf(postingIn("u1", 1, 4), postingIn("u2", 1, 4)),
	)

	assertOrder(
		t,
		documentmatch.HashesOfMostRelevantDocuments(matches, 2, 1),
		[]yacymodel.URLHash{urlHashFor("u1")},
	)
}
