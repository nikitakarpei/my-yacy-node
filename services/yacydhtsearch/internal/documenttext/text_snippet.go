package documenttext

import (
	"strings"
	"unicode"
)

const (
	placeOfTheStartOfTheText = 0
	noWordBoundaryInTheText  = -1
)

func snippetOf(pageText string, placeOfTheFirstQueryWord int, lengthCeiling int) string {
	return textCutAtAWordBoundary(
		pageText[placeOfTheWrittenWordAround(pageText, placeOfTheFirstQueryWord):],
		lengthCeiling,
	)
}

func placeOfTheWrittenWordAround(text string, placeOfTheWord int) int {
	if placeOfTheWord == noQueryWordInTheText {
		return placeOfTheStartOfTheText
	}

	return strings.LastIndexFunc(text[:placeOfTheWord], unicode.IsSpace) + 1
}

func textCutAtAWordBoundary(text string, lengthCeiling int) string {
	placeOfTheCut := placeAfterTheLetters(text, lengthCeiling)
	if placeOfTheCut == len(text) {
		return strings.TrimSpace(text)
	}
	cutText := text[:placeOfTheCut]
	placeOfTheLastWordBoundary := strings.LastIndexFunc(cutText, unicode.IsSpace)
	if placeOfTheLastWordBoundary == noWordBoundaryInTheText ||
		placeOfTheLastWordBoundary == placeOfTheStartOfTheText {
		return strings.TrimSpace(cutText)
	}

	return strings.TrimSpace(cutText[:placeOfTheLastWordBoundary])
}

func placeAfterTheLetters(text string, amountOfLetters int) int {
	lettersPassed := 0
	for place := range text {
		if lettersPassed == amountOfLetters {
			return place
		}
		lettersPassed++
	}

	return len(text)
}
