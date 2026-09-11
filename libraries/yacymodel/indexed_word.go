package yacymodel

import (
	"iter"
	"strings"
	"unicode"
)

const MinimumIndexedWordLength = 2

func WordIsIndexed(word string) bool {
	return len([]rune(word)) >= MinimumIndexedWordLength
}

// WordsIn reads the words this node indexes out of a text: every run of
// letters and digits is a word, and a run too short to index is left out.
func WordsIn(text string) []string {
	var words []string
	for _, word := range PlacedWordsIn(text) {
		words = append(words, word)
	}

	return words
}

// PlacedWordsIn reads the words WordsIn reads, one at a time, each with the
// place in the text where its run of letters and digits starts.
func PlacedWordsIn(text string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		placeOfTheRun := noRunBeingRead
		for place, letter := range text {
			if unicode.IsLetter(letter) || unicode.IsDigit(letter) {
				if placeOfTheRun == noRunBeingRead {
					placeOfTheRun = place
				}

				continue
			}
			if placeOfTheRun != noRunBeingRead &&
				!yieldTheIndexedWord(yield, placeOfTheRun, text[placeOfTheRun:place]) {
				return
			}
			placeOfTheRun = noRunBeingRead
		}
		if placeOfTheRun != noRunBeingRead {
			yieldTheIndexedWord(yield, placeOfTheRun, text[placeOfTheRun:])
		}
	}
}

const noRunBeingRead = -1

func yieldTheIndexedWord(yield func(int, string) bool, place int, run string) bool {
	word := strings.ToLower(run)
	if !WordIsIndexed(word) {
		return true
	}

	return yield(place, word)
}
