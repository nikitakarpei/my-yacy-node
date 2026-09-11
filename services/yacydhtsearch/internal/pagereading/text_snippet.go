package pagereading

import (
	"slices"
	"strings"
	"unicode"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	placeOfTheStartOfTheText = 0
	noWordBoundaryInTheText  = -1
	noWordBeingRead          = -1
)

func snippetOf(
	text string,
	queryWords []yacymodel.Hash,
	lengthCeiling int,
) string {
	placeOfTheSnippet := placeOfTheFirstQueryWordIn(text, queryWords)

	return textCutAtAWordBoundary(text[placeOfTheSnippet:], lengthCeiling)
}

func placeOfTheFirstQueryWordIn(text string, queryWords []yacymodel.Hash) int {
	for _, writtenWord := range writtenWordsIn(text) {
		if !holdsAQueryWord(writtenWord.spelling, queryWords) {
			continue
		}

		return writtenWord.placeInTheText
	}

	return placeOfTheStartOfTheText
}

type writtenWord struct {
	placeInTheText int
	spelling       string
}

func writtenWordsIn(text string) []writtenWord {
	var writtenWords []writtenWord
	placeOfTheWordBeingRead := noWordBeingRead
	for place, letter := range text {
		if !unicode.IsSpace(letter) {
			if placeOfTheWordBeingRead == noWordBeingRead {
				placeOfTheWordBeingRead = place
			}

			continue
		}
		if placeOfTheWordBeingRead == noWordBeingRead {
			continue
		}
		writtenWords = append(writtenWords, writtenWord{
			placeInTheText: placeOfTheWordBeingRead,
			spelling:       text[placeOfTheWordBeingRead:place],
		})
		placeOfTheWordBeingRead = noWordBeingRead
	}
	if placeOfTheWordBeingRead != noWordBeingRead {
		writtenWords = append(writtenWords, writtenWord{
			placeInTheText: placeOfTheWordBeingRead,
			spelling:       text[placeOfTheWordBeingRead:],
		})
	}

	return writtenWords
}

func holdsAQueryWord(spelling string, queryWords []yacymodel.Hash) bool {
	for _, spelledWord := range yacymodel.WordsIn(spelling) {
		if slices.Contains(queryWords, yacymodel.WordHash(spelledWord)) {
			return true
		}
	}

	return false
}

func textCutAtAWordBoundary(text string, lengthCeiling int) string {
	letters := []rune(text)
	if len(letters) <= lengthCeiling {
		return strings.TrimSpace(text)
	}
	cutText := string(letters[:lengthCeiling])
	placeOfTheLastWordBoundary := strings.LastIndexFunc(cutText, unicode.IsSpace)
	if placeOfTheLastWordBoundary == noWordBoundaryInTheText ||
		placeOfTheLastWordBoundary == placeOfTheStartOfTheText {
		return strings.TrimSpace(cutText)
	}

	return strings.TrimSpace(cutText[:placeOfTheLastWordBoundary])
}
