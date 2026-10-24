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
	var found []sentence
	runesBefore := 0
	sentencesOfTheText := sentences.FromString(text)
	for sentencesOfTheText.Next() {
		value := sentencesOfTheText.Value()
		leading := len(value) - len(strings.TrimLeftFunc(value, unicode.IsSpace))
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			start := sentencesOfTheText.Start() + leading
			startRune := runesBefore + utf8.RuneCountInString(value[:leading])
			found = append(found, sentence{
				start:     start,
				end:       start + len(trimmed),
				startRune: startRune,
				endRune:   startRune + utf8.RuneCountInString(trimmed),
			})
		}
		runesBefore += utf8.RuneCountInString(value)
	}
	return withWordsOf(found, text, wordsIn)
}

func withWordsOf(
	found []sentence,
	text string,
	wordsIn func(text string) iter.Seq2[int, string],
) []sentence {
	current := 0
	for place, word := range wordsIn(text) {
		for current < len(found) && place >= found[current].end {
			current++
		}
		if current < len(found) && place >= found[current].start {
			found[current].words = append(found[current].words, placedWord{place, word})
		}
	}
	return found
}
