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
	sentenceTerminals        = ".!?…。！？।؟"
	sentenceClosers          = "\"'”’»)]」』）"
	commaMarks               = "，、،"
	ideographsPerWord        = 2
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
	counts := fieldCountsOf(sentenceText)
	switch {
	case isProse(sentenceText, counts):
		return proseSentence
	case isShortSentence(sentenceText, counts.fields):
		return shortSentence
	default:
		return boilerplateSentence
	}
}

type fieldCounts struct {
	fields          int
	ideographs      int
	casedStarts     int
	lowercaseStarts int
	commas          int
}

func fieldCountsOf(sentenceText string) fieldCounts {
	var counts fieldCounts
	for field := range strings.FieldsSeq(sentenceText) {
		counts.fields++
		firstCharacter := firstCharacterOf(field)
		if unicode.IsUpper(firstCharacter) || unicode.IsLower(firstCharacter) {
			counts.casedStarts++
		}
		if unicode.IsLower(firstCharacter) {
			counts.lowercaseStarts++
		}
		if strings.HasSuffix(field, ",") {
			counts.commas++
		}
		for _, character := range field {
			if unicode.In(
				character,
				unicode.Han,
				unicode.Hiragana,
				unicode.Katakana,
				unicode.Hangul,
			) {
				counts.ideographs++
			}
			if strings.ContainsRune(commaMarks, character) {
				counts.commas++
			}
		}
	}
	return counts
}

func firstCharacterOf(text string) rune {
	firstCharacter, _ := utf8.DecodeRuneInString(text)
	return firstCharacter
}

func isProse(sentenceText string, counts fieldCounts) bool {
	words := counts.fields + counts.ideographs/ideographsPerWord
	if words < proseLeastFields || !endsASentence(sentenceText) {
		return false
	}
	symbols := 0
	for _, character := range sentenceText {
		if strings.ContainsRune(proseSymbols, character) {
			symbols++
		}
	}
	return (counts.casedStarts == 0 ||
		float64(counts.lowercaseStarts) >= proseLeastLowercaseShare*float64(counts.casedStarts)) &&
		float64(counts.commas) <= proseMostCommaShare*float64(words) &&
		float64(symbols) <= proseMostSymbolShare*float64(utf8.RuneCountInString(sentenceText))
}

func endsASentence(sentenceText string) bool {
	lastCharacter, _ := utf8.DecodeLastRuneInString(
		strings.TrimRight(sentenceText, sentenceClosers),
	)
	return strings.ContainsRune(sentenceTerminals, lastCharacter)
}

func isShortSentence(sentenceText string, fields int) bool {
	return unicode.IsLetter(firstCharacterOf(sentenceText)) && endsASentence(sentenceText) &&
		fields <= shortSentenceMostFields
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
