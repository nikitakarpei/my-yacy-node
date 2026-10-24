package textsnippet

import (
	"iter"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/clipperhouse/uax29/v2/sentences"
)

const (
	proseLeastFields         = 6
	proseLeastLowercaseShare = 0.25
	proseMostCommaShare      = 0.25
	proseMostSymbolShare     = 0.04
	proseSymbols             = "#{}[]<>=\\|_*;$@^~`"
	sentenceTerminals        = ".!?…"
	sentenceClosers          = "\"'”’»)]"
	shortSentenceMostFields  = 5
)

type placedWord struct {
	place int
	text  string
}

type sentenceKind int

const (
	boilerplateSentence sentenceKind = iota
	shortSentence
	proseSentence
)

type sentence struct {
	start     int
	end       int
	startRune int
	endRune   int
	kind      sentenceKind
	heading   bool
	words     []placedWord
}

func sentencesIn(text, title string, wordsIn func(text string) iter.Seq2[int, string]) []sentence {
	var foundSentences []sentence
	var collapsedRunes collapsedRuneCounter
	sentencesOfTheText := sentences.FromString(text)
	for sentencesOfTheText.Next() {
		sentenceText := sentencesOfTheText.Value()
		textAfterSpaces := strings.TrimLeftFunc(sentenceText, unicode.IsSpace)
		trimmedText := strings.TrimRightFunc(textAfterSpaces, unicode.IsSpace)
		collapsedRunes.add(sentenceText[:len(sentenceText)-len(textAfterSpaces)])
		if trimmedText != "" {
			start := sentencesOfTheText.Start() + len(sentenceText) - len(textAfterSpaces)
			startRune := collapsedRunes.count
			collapsedRunes.add(trimmedText)
			foundSentences = append(foundSentences, sentence{
				start:     start,
				end:       start + len(trimmedText),
				startRune: startRune,
				endRune:   collapsedRunes.count,
				kind:      kindOf(trimmedText),
			})
		}
		collapsedRunes.add(textAfterSpaces[len(trimmedText):])
	}
	foundSentences = withWordsOf(foundSentences, text, wordsIn)
	return withHeadingsSharing(foundSentences, text, wordSetIn(title, wordsIn))
}

type collapsedRuneCounter struct {
	count        int
	lastWasSpace bool
}

func (counter *collapsedRuneCounter) add(text string) {
	for _, character := range text {
		isSpace := unicode.IsSpace(character)
		if !isSpace || !counter.lastWasSpace {
			counter.count++
		}
		counter.lastWasSpace = isSpace
	}
}

func kindOf(sentenceText string) sentenceKind {
	fields := strings.Fields(sentenceText)
	switch {
	case isProse(sentenceText, fields):
		return proseSentence
	case isShortSentence(sentenceText, fields):
		return shortSentence
	default:
		return boilerplateSentence
	}
}

func isProse(sentenceText string, fields []string) bool {
	if len(fields) < proseLeastFields || !endsASentence(sentenceText) {
		return false
	}
	letterStarts, lowercaseStarts, commas := 0, 0, 0
	for _, field := range fields {
		firstCharacter := firstCharacterOf(field)
		if unicode.IsLetter(firstCharacter) {
			letterStarts++
		}
		if unicode.IsLower(firstCharacter) {
			lowercaseStarts++
		}
		if strings.HasSuffix(field, ",") {
			commas++
		}
	}
	symbols := 0
	for _, character := range sentenceText {
		if strings.ContainsRune(proseSymbols, character) {
			symbols++
		}
	}
	return letterStarts > 0 &&
		float64(lowercaseStarts) >= proseLeastLowercaseShare*float64(letterStarts) &&
		float64(commas) <= proseMostCommaShare*float64(len(fields)) &&
		float64(symbols) <= proseMostSymbolShare*float64(utf8.RuneCountInString(sentenceText))
}

func endsASentence(sentenceText string) bool {
	lastCharacter, _ := utf8.DecodeLastRuneInString(
		strings.TrimRight(sentenceText, sentenceClosers),
	)
	return strings.ContainsRune(sentenceTerminals, lastCharacter)
}

func firstCharacterOf(text string) rune {
	firstCharacter, _ := utf8.DecodeRuneInString(text)
	return firstCharacter
}

func isShortSentence(sentenceText string, fields []string) bool {
	return unicode.IsLetter(firstCharacterOf(sentenceText)) && endsASentence(sentenceText) &&
		len(fields) <= shortSentenceMostFields
}

func withWordsOf(
	foundSentences []sentence,
	text string,
	wordsIn func(text string) iter.Seq2[int, string],
) []sentence {
	currentSentence := 0
	for place, word := range wordsIn(text) {
		for currentSentence < len(foundSentences) && place >= foundSentences[currentSentence].end {
			currentSentence++
		}
		if currentSentence < len(foundSentences) && place >= foundSentences[currentSentence].start {
			foundSentences[currentSentence].words = append(
				foundSentences[currentSentence].words,
				placedWord{place, word},
			)
		}
	}
	return foundSentences
}

func withHeadingsSharing(foundSentences []sentence, text string, titleWords wordSet) []sentence {
	for sentenceIndex, possibleHeading := range foundSentences[:max(0, len(foundSentences)-1)] {
		foundSentences[sentenceIndex].heading = possibleHeading.kind == boilerplateSentence &&
			foundSentences[sentenceIndex+1].kind == proseSentence &&
			unicode.IsUpper(firstCharacterOf(text[possibleHeading.start:])) &&
			titleWords.holdsAnyOf(possibleHeading.words)
	}
	return foundSentences
}

func (edge sentence) isDroppableFor(query askedQuery) bool {
	return edge.kind == boilerplateSentence && !query.holdsAnyOf(edge.words)
}
