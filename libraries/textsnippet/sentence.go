package textsnippet

import (
	"iter"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/clipperhouse/uax29/v2/sentences"
)

type placedWord struct {
	place int
	text  string
}

type sentence struct {
	start     int
	end       int
	startRune int
	endRune   int
	words     []placedWord
}

func sentencesIn(text string, wordsIn func(text string) iter.Seq2[int, string]) []sentence {
	var foundSentences []sentence
	runesBefore := 0
	sentencesOfTheText := sentences.FromString(text)
	for sentencesOfTheText.Next() {
		sentenceText := sentencesOfTheText.Value()
		leadingSpaceBytes := len(
			sentenceText,
		) - len(
			strings.TrimLeftFunc(sentenceText, unicode.IsSpace),
		)
		trimmedText := strings.TrimSpace(sentenceText)
		if trimmedText != "" {
			start := sentencesOfTheText.Start() + leadingSpaceBytes
			startRune := runesBefore + utf8.RuneCountInString(sentenceText[:leadingSpaceBytes])
			foundSentences = append(foundSentences, sentence{
				start:     start,
				end:       start + len(trimmedText),
				startRune: startRune,
				endRune:   startRune + utf8.RuneCountInString(trimmedText),
			})
		}
		runesBefore += utf8.RuneCountInString(sentenceText)
	}
	return withWordsOf(foundSentences, text, wordsIn)
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
