package searchquery_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestTheSpellingOfAQuerySpellsItsWordsExclusionsAndLanguage(t *testing.T) {
	t.Parallel()

	query := searchquery.Query{
		Words:      []string{"reset", "router"},
		Exclusions: []string{"windows"},
		Language:   "lang_en",
	}

	if query.String() != "reset router -windows lr:lang_en" {
		t.Fatalf("String = %q, want reset router -windows lr:lang_en", query.String())
	}
}

func TestWordHashesAddressTheWordsOnTheRing(t *testing.T) {
	t.Parallel()

	query := searchquery.Query{Words: []string{"berlin"}, Exclusions: []string{"rain"}}

	if !slices.Equal(query.WordHashes(), []yacymodel.Hash{yacymodel.WordHash("berlin")}) {
		t.Fatalf("WordHashes = %v, want the hash of berlin", query.WordHashes())
	}
	if !slices.Equal(query.ExclusionHashes(), []yacymodel.Hash{yacymodel.WordHash("rain")}) {
		t.Fatalf("ExclusionHashes = %v, want the hash of rain", query.ExclusionHashes())
	}
}

func TestACompoundWordIsAddressedByItsWordAndCountsForItsParts(t *testing.T) {
	t.Parallel()

	compound := searchquery.CompoundWord{Word: "openstreet", Parts: []string{"open", "street"}}

	if compound.Hash() != yacymodel.WordHash("openstreet") {
		t.Fatalf("Hash = %v, want the hash of openstreet", compound.Hash())
	}
	if !slices.Equal(compound.WordHashes(), hashesOf("open", "street")) {
		t.Fatalf("WordHashes = %v, want the hashes of open and street", compound.WordHashes())
	}
}

func TestTheHashesOfWordsAndCompoundWordsHoldTheWordsThenTheCompoundWordsUpToTheCeiling(
	t *testing.T,
) {
	t.Parallel()

	query := searchquery.Query{
		Words: []string{"open", "street", "map"},
		CompoundWords: []searchquery.CompoundWord{
			{Word: "openstreet", Parts: []string{"open", "street"}},
			{Word: "streetmap", Parts: []string{"street", "map"}},
		},
	}

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
