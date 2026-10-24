package textsnippet

import (
	"iter"
	"unicode"
	"unicode/utf8"
)

type passage struct {
	sentences []sentence
}

func (candidate passage) words() iter.Seq[placedWord] {
	return func(yield func(placedWord) bool) {
		for _, sentenceOfThePassage := range candidate.sentences {
			for _, word := range sentenceOfThePassage.words {
				if !yield(word) {
					return
				}
			}
		}
	}
}

func (candidate passage) textWithin(text string, query askedQuery, lengthCeiling int) string {
	if len(candidate.sentences) == 0 {
		return ""
	}
	first, last := candidate.sentences[0], candidate.sentences[len(candidate.sentences)-1]
	if last.endRune-first.startRune <= lengthCeiling {
		return text[first.start:last.end]
	}
	return windowAround(
		text[first.start:first.end],
		query.firstMatchIn(candidate)-first.start,
		lengthCeiling,
	)
}

func windowAround(sentenceText string, placeOfTheMatch int, lengthCeiling int) string {
	letters := []rune(sentenceText)
	runeOfTheMatch := utf8.RuneCountInString(sentenceText[:placeOfTheMatch])
	windowStart := max(0, runeOfTheMatch-lengthCeiling/2)
	windowEnd := min(len(letters), windowStart+lengthCeiling)
	windowStart = max(0, windowEnd-lengthCeiling)
	if windowStart > 0 {
		windowStart = startOfAWordFrom(letters, windowStart, windowEnd)
	}
	if windowEnd < len(letters) {
		windowEnd = endOfAWordBefore(letters, windowStart, windowEnd)
	}
	return string(letters[windowStart:windowEnd])
}

func startOfAWordFrom(letters []rune, windowStart, windowEnd int) int {
	if unicode.IsSpace(letters[windowStart-1]) {
		return windowStart
	}
	for place := windowStart; place < windowEnd; place++ {
		if unicode.IsSpace(letters[place]) {
			return place + 1
		}
	}
	return windowStart
}

func endOfAWordBefore(letters []rune, windowStart, windowEnd int) int {
	if unicode.IsSpace(letters[windowEnd]) {
		return windowEnd
	}
	for place := windowEnd - 1; place > windowStart; place-- {
		if unicode.IsSpace(letters[place]) {
			return place
		}
	}
	return windowEnd
}
