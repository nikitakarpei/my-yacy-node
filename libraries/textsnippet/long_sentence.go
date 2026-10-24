package textsnippet

import (
	"sort"
	"unicode"
	"unicode/utf8"
)

const (
	windowLeadDivisor = 4
	cutMark           = "…"
)

type longSentence struct {
	letters []rune
	words   []placedWord
}

func longSentenceOf(text string, cutSentence sentence) longSentence {
	var collapsedRunes collapsedRuneCounter
	placeBefore := cutSentence.start
	words := make([]placedWord, 0, len(cutSentence.words))
	for _, word := range cutSentence.words {
		collapsedRunes.add(text[placeBefore:word.place])
		placeBefore = word.place
		words = append(words, placedWord{place: collapsedRunes.count, text: word.text})
	}
	return longSentence{
		letters: []rune(withSpacesCollapsed(text[cutSentence.start:cutSentence.end])),
		words:   words,
	}
}

func (cut longSentence) windowFor(query askedQuery, lengthCeiling int) string {
	windowStart, windowEnd := cut.windowFrom(0, lengthCeiling)
	var bestHits queryHits
	for _, anchor := range cut.words {
		if !query.holds(anchor.text) {
			continue
		}
		anchorStart, anchorEnd := cut.windowFrom(
			anchor.place-lengthCeiling/windowLeadDivisor,
			lengthCeiling,
		)
		hits := query.hitsAmong(cut.wordsBetween(anchorStart, anchorEnd))
		if hits.outweigh(bestHits) {
			windowStart, windowEnd, bestHits = anchorStart, anchorEnd, hits
		}
	}
	return cut.textBetween(windowStart, windowEnd)
}

func (cut longSentence) windowFrom(windowStart, lengthCeiling int) (int, int) {
	windowEnd := min(len(cut.letters), max(0, windowStart)+lengthCeiling)
	return max(0, windowEnd-lengthCeiling), windowEnd
}

func (cut longSentence) wordsBetween(windowStart, windowEnd int) []placedWord {
	first := sort.Search(
		len(cut.words),
		func(place int) bool { return cut.words[place].place >= windowStart },
	)
	last := sort.Search(
		len(cut.words),
		func(place int) bool { return cut.words[place].place >= windowEnd },
	)
	return cut.words[first:last]
}

func (cut longSentence) textBetween(windowStart, windowEnd int) string {
	cutMarkLength := utf8.RuneCountInString(cutMark) + 1
	leadingMark, trailingMark := "", ""
	if windowStart > 0 {
		windowStart = cut.startOfAWordFrom(min(windowStart+cutMarkLength, windowEnd-1), windowEnd)
		leadingMark = cutMark + " "
	}
	if windowEnd < len(cut.letters) {
		windowEnd = cut.endOfAWordBefore(windowStart, max(windowStart+1, windowEnd-cutMarkLength))
		trailingMark = " " + cutMark
	}
	return leadingMark + string(cut.letters[windowStart:windowEnd]) + trailingMark
}

func (cut longSentence) startOfAWordFrom(windowStart, windowEnd int) int {
	if unicode.IsSpace(cut.letters[windowStart-1]) {
		return windowStart
	}
	for place := windowStart; place < windowEnd; place++ {
		if unicode.IsSpace(cut.letters[place]) {
			return place + 1
		}
	}
	return windowStart
}

func (cut longSentence) endOfAWordBefore(windowStart, windowEnd int) int {
	if unicode.IsSpace(cut.letters[windowEnd]) {
		return windowEnd
	}
	for place := windowEnd - 1; place > windowStart; place-- {
		if unicode.IsSpace(cut.letters[place]) {
			return place
		}
	}
	return windowEnd
}
