package yacymodel_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestAWordOfTwoLettersIsIndexed(t *testing.T) {
	t.Parallel()

	if !yacymodel.WordIsIndexed("go") {
		t.Fatal("WordIsIndexed(go) = false, want true")
	}
}

func TestAWordOfOneLetterIsNotIndexed(t *testing.T) {
	t.Parallel()

	if yacymodel.WordIsIndexed("1") {
		t.Fatal("WordIsIndexed(1) = true, want false")
	}
}

func TestATextIsReadAsTheWordsBetweenItsPunctuation(t *testing.T) {
	t.Parallel()

	words := yacymodel.WordsIn("Terraform vs. OpenTofu: what changed?")

	want := []string{"terraform", "vs", "opentofu", "what", "changed"}
	if !slices.Equal(words, want) {
		t.Fatalf("WordsIn = %v, want %v", words, want)
	}
}

func TestAWordTooShortToIndexIsNoWordOfAText(t *testing.T) {
	t.Parallel()

	words := yacymodel.WordsIn("a berlin 1")

	if !slices.Equal(words, []string{"berlin"}) {
		t.Fatalf("WordsIn = %v, want only the indexed word", words)
	}
}

func TestATextOfNoWordsReadsAsNoWord(t *testing.T) {
	t.Parallel()

	if words := yacymodel.WordsIn(" -- "); len(words) != 0 {
		t.Fatalf("WordsIn = %v, want no word", words)
	}
}

func TestAWordOfOneMultibyteLetterIsNotIndexed(t *testing.T) {
	t.Parallel()

	if yacymodel.WordIsIndexed("ä") {
		t.Fatal("WordIsIndexed(ä) = true, want false")
	}
}

func TestEachWordOfATextIsReadWithThePlaceWhereItStarts(t *testing.T) {
	t.Parallel()

	placePerWord := map[string]int{}
	for place, word := range yacymodel.PlacedWordsIn("Über Berlin: a wall") {
		placePerWord[word] = place
	}

	want := map[string]int{"über": 0, "berlin": 6, "wall": 16}
	if !maps.Equal(placePerWord, want) {
		t.Fatalf("PlacedWordsIn = %v, want %v", placePerWord, want)
	}
}

func TestTheWordsOfATextStopWhenTheReaderStops(t *testing.T) {
	t.Parallel()

	var wordsRead []string
	for _, word := range yacymodel.PlacedWordsIn("berlin holds a wall") {
		wordsRead = append(wordsRead, word)
		if len(wordsRead) == 2 {
			break
		}
	}

	if !slices.Equal(wordsRead, []string{"berlin", "holds"}) {
		t.Fatalf("PlacedWordsIn read %v, want the first two words", wordsRead)
	}
}
