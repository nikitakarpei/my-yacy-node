package yacymodel

import (
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
	runs := strings.FieldsFunc(text, func(letter rune) bool {
		return !unicode.IsLetter(letter) && !unicode.IsDigit(letter)
	})

	words := make([]string, 0, len(runs))
	for _, run := range runs {
		word := strings.ToLower(run)
		if !WordIsIndexed(word) {
			continue
		}
		words = append(words, word)
	}

	return words
}
