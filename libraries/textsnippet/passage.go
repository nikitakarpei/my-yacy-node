package textsnippet

import (
	"iter"
	"unicode"
	"unicode/utf8"
)

type passage struct {
	sentences []sentence
}

func (candidate passage) words() iter.Seq2[int, placedWord] {
	return func(yield func(int, placedWord) bool) {
		place := 0
		for _, sentenceOfThePassage := range candidate.sentences {
			for _, word := range sentenceOfThePassage.words {
				if !yield(place, word) {
					return
				}
				place++
			}
		}
	}
}

func (candidate passage) textWithin(text string, asked query, lengthCeiling int) string {
	if len(candidate.sentences) == 0 {
		return ""
	}
	first, last := candidate.sentences[0], candidate.sentences[len(candidate.sentences)-1]
	if last.endRune-first.startRune <= lengthCeiling {
		return text[first.start:last.end]
	}
	return windowAround(
		text[first.start:first.end],
		asked.firstPlaceIn(candidate)-first.start,
		lengthCeiling,
	)
}

func windowAround(sentenceText string, placeOfTheMatch int, lengthCeiling int) string {
	letters := []rune(sentenceText)
	runeOfTheMatch := utf8.RuneCountInString(sentenceText[:placeOfTheMatch])
	from := max(0, runeOfTheMatch-lengthCeiling/2)
	to := min(len(letters), from+lengthCeiling)
	from = max(0, to-lengthCeiling)
	if from > 0 {
		from = startOfAWordFrom(letters, from, to)
	}
	if to < len(letters) {
		to = endOfAWordBefore(letters, from, to)
	}
	return string(letters[from:to])
}

func startOfAWordFrom(letters []rune, from, to int) int {
	if unicode.IsSpace(letters[from-1]) {
		return from
	}
	for place := from; place < to; place++ {
		if unicode.IsSpace(letters[place]) {
			return place + 1
		}
	}
	return from
}

func endOfAWordBefore(letters []rune, from, to int) int {
	if unicode.IsSpace(letters[to]) {
		return to
	}
	for place := to - 1; place > from; place-- {
		if unicode.IsSpace(letters[place]) {
			return place
		}
	}
	return to
}
