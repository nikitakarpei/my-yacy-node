package featurehashing_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spammodel/featurehashing"
)

func TestCharacterNgramsStayInsideLowercasedPaddedWords(t *testing.T) {
	ngrams := featurehashing.CharacterNgramsFrom(
		"Cheap  Pills a\tBIG",
		featurehashing.NgramSizes{Shortest: 4, Longest: 4},
	)

	want := []string{
		" che",
		"chea",
		"heap",
		"eap ",
		" pil",
		"pill",
		"ills",
		"lls ",
		" a ",
		" big",
		"big ",
	}
	if !slices.Equal(ngrams, want) {
		t.Errorf("CharacterNgramsFrom = %q, want %q", ngrams, want)
	}
}

func TestCharacterNgramsOfAShortWordAreTheWholeWordOnce(t *testing.T) {
	ngrams := featurehashing.CharacterNgramsFrom(
		"ab cdef",
		featurehashing.NgramSizes{Shortest: 3, Longest: 5},
	)

	want := []string{
		" ab", "ab ", " ab ",
		" cd", "cde", "def", "ef ", " cde", "cdef", "def ", " cdef", "cdef ",
	}
	if !slices.Equal(ngrams, want) {
		t.Errorf("CharacterNgramsFrom = %q, want %q", ngrams, want)
	}
}

func TestWordNgramsJoinNeighbouringLowercasedWords(t *testing.T) {
	ngrams := featurehashing.WordNgramsFrom(
		"Div P div",
		featurehashing.NgramSizes{Shortest: 1, Longest: 2},
	)

	want := []string{"div", "p", "div", "div p", "p div"}
	if !slices.Equal(ngrams, want) {
		t.Errorf("WordNgramsFrom = %q, want %q", ngrams, want)
	}
}

func TestWordNgramsLongerThanTheDocumentAreAbsent(t *testing.T) {
	ngrams := featurehashing.WordNgramsFrom(
		"one",
		featurehashing.NgramSizes{Shortest: 2, Longest: 2},
	)

	if len(ngrams) != 0 {
		t.Errorf("WordNgramsFrom = %q, want none", ngrams)
	}
}

func TestHashedCharacterNgramsMatchScikitLearn(t *testing.T) {
	entries := featurehashing.EntriesFrom(
		featurehashing.CharacterNgramsFrom(
			"Cheap cheap",
			featurehashing.NgramSizes{Shortest: 4, Longest: 4},
		),
	)

	want := []featurehashing.Entry{
		{Index: 275445, Value: 0.5},
		{Index: 319195, Value: 0.5},
		{Index: 436188, Value: 0.5},
		{Index: 945087, Value: 0.5},
	}
	if !slices.Equal(entries, want) {
		t.Errorf("EntriesFrom = %v, want %v", entries, want)
	}
}
