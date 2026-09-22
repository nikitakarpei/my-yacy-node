package searchquery_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestQueryFromKeepsEachSpokenWordOnce(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom(`Berlin  "Weather" berlin +forecast`, "")

	if !slices.Equal(query.Words, []string{"berlin", "weather", "forecast"}) {
		t.Fatalf("Words = %v, want berlin weather forecast", query.Words)
	}
}

func TestQueryFromPutsAMinusWordUnderExclusions(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("berlin -rain", "")

	if !slices.Equal(query.Words, []string{"berlin"}) {
		t.Fatalf("Words = %v, want berlin", query.Words)
	}
	if !slices.Equal(query.Exclusions, []string{"rain"}) {
		t.Fatalf("Exclusions = %v, want rain", query.Exclusions)
	}
}

func TestQueryFromDropsAWordTooShortForAnIndex(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("berlin 1", "")

	if !slices.Equal(query.Words, []string{"berlin"}) {
		t.Fatalf("Words = %v, want berlin alone", query.Words)
	}
}

func TestQueryFromDropsAWordTooShortForAnIndexWhenItStandsAlone(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("1", "")

	if len(query.Words) != 0 {
		t.Fatalf("Words = %v, want no word", query.Words)
	}
}

func TestQueryFromDropsAnExclusionTooShortForAnIndex(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("berlin -a", "")

	if len(query.Exclusions) != 0 {
		t.Fatalf("Exclusions = %v, want no exclusion", query.Exclusions)
	}
}

func TestQueryFromReadsNothingOutOfPunctuationAlone(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom(`- "" +`, "")

	if len(query.Words) != 0 || len(query.Exclusions) != 0 {
		t.Fatalf("QueryFrom = %+v, want no words and no exclusions", query)
	}
}

func TestQueryFromLeavesTheStopwordsOfTheQueryLanguageOut(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("how do i reset my router", "")

	if !slices.Equal(query.Words, []string{"reset", "router"}) {
		t.Fatalf("Words = %v, want reset router", query.Words)
	}
}

func TestQueryFromKeepsAStopwordAmongTheExclusions(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("how to install debian -the", "")

	if !slices.Equal(query.Words, []string{"install", "debian"}) {
		t.Fatalf("Words = %v, want install debian", query.Words)
	}
	if !slices.Equal(query.Exclusions, []string{"the"}) {
		t.Fatalf("Exclusions = %v, want the", query.Exclusions)
	}
}

func TestTheSpellingOfAQuerySpellsTheWordsItKept(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("how do i reset my router -windows", "lang_en")

	if query.String() != "reset router -windows lr:lang_en" {
		t.Fatalf("String = %q, want reset router -windows lr:lang_en", query.String())
	}
}

func TestWordHashesAddressTheWordsOnTheRing(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("berlin -rain", "")

	if !slices.Equal(query.WordHashes(), []yacymodel.Hash{yacymodel.WordHash("berlin")}) {
		t.Fatalf("WordHashes = %v, want the hash of berlin", query.WordHashes())
	}
	if !slices.Equal(query.ExclusionHashes(), []yacymodel.Hash{yacymodel.WordHash("rain")}) {
		t.Fatalf("ExclusionHashes = %v, want the hash of rain", query.ExclusionHashes())
	}
}

func TestTheCompoundWordsOfAQueryJoinItsSpokenNeighboursInPairsThenTriples(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("open street map", "")

	want := []searchquery.CompoundWord{
		{Hash: yacymodel.WordHash("openstreet"), WordHashes: hashesOf("open", "street")},
		{Hash: yacymodel.WordHash("streetmap"), WordHashes: hashesOf("street", "map")},
		{Hash: yacymodel.WordHash("openstreetmap"), WordHashes: hashesOf("open", "street", "map")},
	}
	if !slices.EqualFunc(query.CompoundWords, want, compoundWordsEqual) {
		t.Fatalf("CompoundWords = %v, want %v", query.CompoundWords, want)
	}
}

func TestADroppedStopwordBreaksACompoundWord(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("yacy peer to peer search", "en")

	want := []searchquery.CompoundWord{
		{Hash: yacymodel.WordHash("yacypeer"), WordHashes: hashesOf("yacy", "peer")},
		{Hash: yacymodel.WordHash("peersearch"), WordHashes: hashesOf("peer", "search")},
	}
	if !slices.EqualFunc(query.CompoundWords, want, compoundWordsEqual) {
		t.Fatalf("CompoundWords = %v, want %v", query.CompoundWords, want)
	}
}

func TestAQueryOfOneWordHasNoCompoundWord(t *testing.T) {
	t.Parallel()

	if compounds := searchquery.QueryFrom("berlin", "").CompoundWords; len(compounds) != 0 {
		t.Fatalf("CompoundWords = %v, want none", compounds)
	}
}

func TestTheHashesOfWordsAndCompoundWordsHoldTheWordsThenTheCompoundWordsUpToTheCeiling(
	t *testing.T,
) {
	t.Parallel()

	query := searchquery.QueryFrom("open street map", "")

	want := append(hashesOf("open", "street", "map"), yacymodel.WordHash("openstreet"))
	if got := query.HashesOfWordsAndCompoundWordsUpTo(1); !slices.Equal(got, want) {
		t.Fatalf("HashesOfWordsAndCompoundWordsUpTo(1) = %v, want %v", got, want)
	}
}

func hashesOf(words ...string) []yacymodel.Hash {
	hashes := make([]yacymodel.Hash, 0, len(words))
	for _, word := range words {
		hashes = append(hashes, yacymodel.WordHash(word))
	}

	return hashes
}

func compoundWordsEqual(one, other searchquery.CompoundWord) bool {
	return one.Hash == other.Hash && slices.Equal(one.WordHashes, other.WordHashes)
}
