package queryreading_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func TestTheCompoundWordsOfAQueryJoinItsSpokenNeighboursInPairsThenTriples(t *testing.T) {
	t.Parallel()

	query := queryreading.QueryFrom("open street map", "")

	want := []searchquery.CompoundWord{
		{Word: "openstreet", Parts: []string{"open", "street"}},
		{Word: "streetmap", Parts: []string{"street", "map"}},
		{Word: "openstreetmap", Parts: []string{"open", "street", "map"}},
	}
	if !slices.EqualFunc(query.CompoundWords, want, compoundWordsEqual) {
		t.Fatalf("CompoundWords = %v, want %v", query.CompoundWords, want)
	}
}

func TestADroppedStopwordBreaksACompoundWord(t *testing.T) {
	t.Parallel()

	query := queryreading.QueryFrom("yacy peer to peer search", "en")

	want := []searchquery.CompoundWord{
		{Word: "yacypeer", Parts: []string{"yacy", "peer"}},
		{Word: "peersearch", Parts: []string{"peer", "search"}},
	}
	if !slices.EqualFunc(query.CompoundWords, want, compoundWordsEqual) {
		t.Fatalf("CompoundWords = %v, want %v", query.CompoundWords, want)
	}
}

func TestAQueryOfOneWordHasNoCompoundWord(t *testing.T) {
	t.Parallel()

	if compounds := queryreading.QueryFrom("berlin", "").CompoundWords; len(compounds) != 0 {
		t.Fatalf("CompoundWords = %v, want none", compounds)
	}
}

func compoundWordsEqual(one, other searchquery.CompoundWord) bool {
	return one.Word == other.Word && slices.Equal(one.Parts, other.Parts)
}
