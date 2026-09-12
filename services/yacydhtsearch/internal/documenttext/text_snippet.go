package documenttext

import (
	"strings"
	"unicode"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	placeOfTheStartOfTheText = 0
	noWordBoundaryInTheText  = -1
)

func snippetOf(pageText string, queryWords []yacymodel.Hash, lengthCeiling int) string {
	bestPassage := bestPassageForTheQueryWords(queryWords, pageText, lengthCeiling)

	return textCutAtAWordBoundary(strings.TrimSpace(bestPassage.text), lengthCeiling)
}

func textCutAtAWordBoundary(text string, lengthCeiling int) string {
	placeOfTheCut := placeAfterTheLetters(text, lengthCeiling)
	if placeOfTheCut == len(text) {
		return text
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
