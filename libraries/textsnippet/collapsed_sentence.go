package textsnippet

import (
	"math"
	"sort"
	"unicode"
	"unicode/utf8"
)

const (
	windowLeadShare = 0.25
	cutMark         = "…"
	leadingCutMark  = cutMark + " "
	trailingCutMark = " " + cutMark
)

type collapsedWord struct {
	startRune int
	text      string
}

type collapsedSentence struct {
	characters []rune
	words      []collapsedWord
}

func collapsedSentenceOf(text string, overlongSentence sentence) collapsedSentence {
	var collapsedRunes collapsedRuneCounter
	placeBefore := overlongSentence.start
	words := make([]collapsedWord, 0, len(overlongSentence.words))
	for _, word := range overlongSentence.words {
		collapsedRunes.add(text[placeBefore:word.place])
		placeBefore = word.place
		words = append(words, collapsedWord{startRune: collapsedRunes.count, text: word.text})
	}
	return collapsedSentence{
		characters: []rune(withSpacesCollapsed(text[overlongSentence.start:overlongSentence.end])),
		words:      words,
	}
}

func (collapsed collapsedSentence) textFor(query askedQuery, lengthCeiling int) string {
	windowStart, windowEnd := collapsed.windowBoundsFrom(0, lengthCeiling)
	windowLead := int(math.Round(float64(lengthCeiling) * windowLeadShare))
	var bestHits queryHits
	for _, anchor := range collapsed.words {
		if !query.holds(anchor.text) {
			continue
		}
		anchoredStart, anchoredEnd := collapsed.windowBoundsFrom(
			anchor.startRune-windowLead,
			lengthCeiling,
		)
		hits := collapsed.hitsBetween(anchoredStart, anchoredEnd, query)
		if hits.outweigh(bestHits) {
			windowStart, windowEnd, bestHits = anchoredStart, anchoredEnd, hits
		}
	}
	return collapsed.markedTextBetween(windowStart, windowEnd)
}

func (collapsed collapsedSentence) windowBoundsFrom(windowStart, lengthCeiling int) (int, int) {
	windowEnd := min(len(collapsed.characters), max(0, windowStart)+lengthCeiling)
	return max(0, windowEnd-lengthCeiling), windowEnd
}

func (collapsed collapsedSentence) hitsBetween(
	windowStart, windowEnd int,
	query askedQuery,
) queryHits {
	counter := hitCounterFor(query)
	firstWordInWindow := sort.Search(len(collapsed.words), func(wordIndex int) bool {
		return collapsed.words[wordIndex].startRune >= windowStart
	})
	for _, word := range collapsed.words[firstWordInWindow:] {
		if word.startRune >= windowEnd {
			break
		}
		counter.countWord(word.text)
	}
	return counter.hits()
}

func (collapsed collapsedSentence) markedTextBetween(windowStart, windowEnd int) string {
	cutMarkLength := utf8.RuneCountInString(leadingCutMark)
	leadingMark, trailingMark := "", ""
	if windowStart > 0 {
		windowStart = collapsed.startOfAWordFrom(
			min(windowStart+cutMarkLength, windowEnd-1),
			windowEnd,
		)
		leadingMark = leadingCutMark
	}
	if windowEnd < len(collapsed.characters) {
		windowEnd = collapsed.endOfAWordBefore(
			max(windowStart+1, windowEnd-cutMarkLength),
			windowStart,
		)
		trailingMark = trailingCutMark
	}
	return leadingMark + string(collapsed.characters[windowStart:windowEnd]) + trailingMark
}

func (collapsed collapsedSentence) startOfAWordFrom(characterIndex, highestIndex int) int {
	if unicode.IsSpace(collapsed.characters[characterIndex-1]) {
		return characterIndex
	}
	for spaceIndex := characterIndex; spaceIndex < highestIndex; spaceIndex++ {
		if unicode.IsSpace(collapsed.characters[spaceIndex]) {
			return spaceIndex + 1
		}
	}
	return characterIndex
}

func (collapsed collapsedSentence) endOfAWordBefore(characterIndex, lowestIndex int) int {
	if unicode.IsSpace(collapsed.characters[characterIndex]) {
		return characterIndex
	}
	for spaceIndex := characterIndex - 1; spaceIndex > lowestIndex; spaceIndex-- {
		if unicode.IsSpace(collapsed.characters[spaceIndex]) {
			return spaceIndex
		}
	}
	return characterIndex
}
