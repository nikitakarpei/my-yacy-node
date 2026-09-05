package documentmatch_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/documentmatch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
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

func matchOf(postings ...yacymodel.RWIPosting) termpostings.Match {
	byDocument := make(map[yacymodel.URLHash]yacymodel.RWIPosting, len(postings))
	for _, posting := range postings {
		byDocument[posting.URLHash] = posting
	}

	return termpostings.Match{PostingPerDocument: byDocument, PostingsHeld: len(postings)}
}

func postingIn(url string, occurrences, textPosition int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		URLHash:      urlHashFor(url),
		Hits:         occurrences,
		TextPosition: textPosition,
	}
}

func matchesOfTwoTerms(
	first, second termpostings.Match,
) map[yacymodel.URLHash]documentmatch.Match {
	firstTerm, secondTerm := hashFor("w1"), hashFor("w2")

	return documentmatch.MatchesAcrossEveryTerm(
		[]yacymodel.Hash{firstTerm, secondTerm},
		map[yacymodel.Hash]termpostings.Match{firstTerm: first, secondTerm: second},
	)
}

func assertOrder(t *testing.T, ordered []yacymodel.RWIPosting, want []yacymodel.URLHash) {
	t.Helper()

	if len(ordered) != len(want) {
		t.Fatalf("order = %v, want %v", ordered, want)
	}
	for position, documentHash := range want {
		if ordered[position].URLHash != documentHash {
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
		documentmatch.PostingsOfMostRelevantDocuments(matches, 2, 0),
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
		documentmatch.PostingsOfMostRelevantDocuments(matches, 2, 0),
		[]yacymodel.URLHash{urlHashFor("u1"), urlHashFor("u2")},
	)
}

func TestMatchesAcrossEveryTermKeepsTheLargestCountAndEarliestPosition(t *testing.T) {
	matches := matchesOfTwoTerms(
		matchOf(postingIn("u1", 2, 40)),
		matchOf(postingIn("u1", 5, 9)),
	)

	postings := documentmatch.PostingsOfMostRelevantDocuments(matches, 2, 0)
	if len(postings) != 1 {
		t.Fatalf("postings = %d, want 1", len(postings))
	}
	if postings[0].Hits != 5 {
		t.Errorf("hits = %d, want 5", postings[0].Hits)
	}
	if postings[0].TextPosition != 9 {
		t.Errorf("text position = %d, want 9", postings[0].TextPosition)
	}
}

func TestMatchesAcrossEveryTermIgnoresAnAbsentPosition(t *testing.T) {
	matches := matchesOfTwoTerms(
		matchOf(postingIn("u1", 1, 0)),
		matchOf(postingIn("u1", 1, 12)),
	)

	postings := documentmatch.PostingsOfMostRelevantDocuments(matches, 2, 0)
	if len(postings) != 1 || postings[0].TextPosition != 12 {
		t.Fatalf("postings = %+v, want one at position 12", postings)
	}
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
		documentmatch.PostingsOfMostRelevantDocuments(within, 2, 0),
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
		documentmatch.PostingsOfMostRelevantDocuments(within, 2, 0),
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
		documentmatch.PostingsOfMostRelevantDocuments(matches, 2, 0),
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
		documentmatch.PostingsOfMostRelevantDocuments(matches, 2, 1),
		[]yacymodel.URLHash{urlHashFor("u1")},
	)
}
