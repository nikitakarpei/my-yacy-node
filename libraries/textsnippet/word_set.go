package textsnippet

import "iter"

type wordSet map[string]struct{}

func wordSetIn(text string, wordsIn func(text string) iter.Seq2[int, string]) wordSet {
	words := wordSet{}
	for _, word := range wordsIn(text) {
		words[word] = struct{}{}
	}
	return words
}

func (words wordSet) holds(word string) bool {
	_, isHeld := words[word]
	return isHeld
}

func (words wordSet) holdsAnyOf(placedWords []placedWord) bool {
	for _, word := range placedWords {
		if words.holds(word.text) {
			return true
		}
	}
	return false
}
