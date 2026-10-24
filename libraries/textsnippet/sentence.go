package textsnippet

import (
	"iter"
	"strings"
	"unicode"

	"github.com/clipperhouse/uax29/v2/sentences"
)

const (
	proseLeastWords          = 6
	proseLeastLowercaseShare = 0.25
	proseMostCommaShare      = 0.25
	proseMostSymbolShare     = 0.04
	proseSymbols             = "#{}[]<>=\\|_*;$@^~`"
	sentenceTerminals        = ".!?…"
	sentenceClosers          = "\"'”’»)]"
	shortSentenceMostWords   = 5
)

type placedWord struct {
	place int
	text  string
}

type sentence struct {
	start       int
	end         int
	startRune   int
	endRune     int
	prose       bool
	boilerplate bool
	heading     bool
	words       []placedWord
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
				start:       start,
				end:         start + len(trimmedText),
				startRune:   startRune,
				endRune:     collapsedRunes.count,
				prose:       isProse(trimmedText),
				boilerplate: !isProse(trimmedText) && !isShortSentence(trimmedText),
			})
		}
		collapsedRunes.add(textAfterSpaces[len(trimmedText):])
	}
	return withHeadingsFor(withWordsOf(foundSentences, text, wordsIn), text, title, wordsIn)
}

type collapsedRuneCounter struct {
	count        int
	lastWasSpace bool
}

func (counter *collapsedRuneCounter) add(text string) {
	for _, letter := range text {
		isSpace := unicode.IsSpace(letter)
		if !isSpace || !counter.lastWasSpace {
			counter.count++
		}
		counter.lastWasSpace = isSpace
	}
}

func isProse(sentenceText string) bool {
	words := strings.Fields(sentenceText)
	if len(words) < proseLeastWords || !endsASentence(sentenceText) {
		return false
	}
	letterStarts, lowercaseStarts, commas := 0, 0, 0
	for _, word := range words {
		firstLetter := []rune(word)[0]
		if unicode.IsLetter(firstLetter) {
			letterStarts++
		}
		if unicode.IsLower(firstLetter) {
			lowercaseStarts++
		}
		if strings.HasSuffix(word, ",") {
			commas++
		}
	}
	symbols := 0
	for _, letter := range sentenceText {
		if strings.ContainsRune(proseSymbols, letter) {
			symbols++
		}
	}
	return letterStarts > 0 &&
		float64(lowercaseStarts) >= proseLeastLowercaseShare*float64(letterStarts) &&
		float64(commas) <= proseMostCommaShare*float64(len(words)) &&
		float64(symbols) <= proseMostSymbolShare*float64(len([]rune(sentenceText)))
}

func endsASentence(sentenceText string) bool {
	ending := []rune(strings.TrimRight(sentenceText, sentenceClosers))
	return len(ending) > 0 && strings.ContainsRune(sentenceTerminals, ending[len(ending)-1])
}

func isShortSentence(sentenceText string) bool {
	return unicode.IsLetter([]rune(sentenceText)[0]) && endsASentence(sentenceText) &&
		len(strings.Fields(sentenceText)) <= shortSentenceMostWords
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

func withHeadingsFor(
	foundSentences []sentence,
	text, title string,
	wordsIn func(text string) iter.Seq2[int, string],
) []sentence {
	titleWords := map[string]struct{}{}
	for _, titleWord := range wordsIn(title) {
		titleWords[titleWord] = struct{}{}
	}
	for place := range foundSentences[:max(0, len(foundSentences)-1)] {
		candidate := foundSentences[place]
		foundSentences[place].heading = candidate.boilerplate && foundSentences[place+1].prose &&
			unicode.IsUpper([]rune(text[candidate.start:candidate.end])[0]) &&
			holdsAnyOf(candidate.words, titleWords)
	}
	return foundSentences
}

func holdsAnyOf(words []placedWord, chosenWords map[string]struct{}) bool {
	for _, word := range words {
		if _, isChosen := chosenWords[word.text]; isChosen {
			return true
		}
	}
	return false
}
