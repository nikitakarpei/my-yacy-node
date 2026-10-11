// Package stopwords holds the Snowball stopword lists of English, German,
// French, Spanish, Italian and Russian, and names the one language whose list
// covers the most of a set of words.
package stopwords

import (
	_ "embed"
	"slices"
	"strings"
)

type Stopwords struct {
	words map[string]struct{}
}

func Languages() []string {
	languages := make([]string, 0, len(stopwordsPerLanguage))
	for language := range stopwordsPerLanguage {
		languages = append(languages, language)
	}
	slices.Sort(languages)

	return languages
}

func Of(language string) (Stopwords, bool) {
	stopwordsOfTheLanguage, listed := stopwordsPerLanguage[language]

	return stopwordsOfTheLanguage, listed
}

func (s Stopwords) Contains(word string) bool {
	_, contained := s.words[word]

	return contained
}

func (s Stopwords) Words() []string {
	words := make([]string, 0, len(s.words))
	for word := range s.words {
		words = append(words, word)
	}
	slices.Sort(words)

	return words
}

func LanguageCoveringMostOf(words []string) (string, bool) {
	languageOfTheMostCoveredWords := ""
	mostCoveredWords := 0
	amountOfLanguagesCoveringAsMany := 0
	for language, stopwordsOfTheLanguage := range stopwordsPerLanguage {
		coveredWords := stopwordsOfTheLanguage.amountOfCoveredWordsIn(words)
		switch {
		case coveredWords > mostCoveredWords:
			languageOfTheMostCoveredWords = language
			mostCoveredWords = coveredWords
			amountOfLanguagesCoveringAsMany = 1
		case coveredWords == mostCoveredWords:
			amountOfLanguagesCoveringAsMany++
		}
	}
	if mostCoveredWords == 0 || amountOfLanguagesCoveringAsMany > 1 {
		return "", false
	}

	return languageOfTheMostCoveredWords, true
}

func (s Stopwords) amountOfCoveredWordsIn(words []string) int {
	amountOfCoveredWords := 0
	for _, word := range words {
		if s.Contains(word) {
			amountOfCoveredWords++
		}
	}

	return amountOfCoveredWords
}

var stopwordsPerLanguage = map[string]Stopwords{
	"de": stopwordsInTheList(germanStopwordList),
	"en": stopwordsInTheList(englishStopwordList),
	"es": stopwordsInTheList(spanishStopwordList),
	"fr": stopwordsInTheList(frenchStopwordList),
	"it": stopwordsInTheList(italianStopwordList),
	"ru": stopwordsInTheList(russianStopwordList),
}

func stopwordsInTheList(list string) Stopwords {
	words := make(map[string]struct{})
	for _, word := range strings.Fields(list) {
		words[word] = struct{}{}
	}

	return Stopwords{words: words}
}

//go:embed lists/german.txt
var germanStopwordList string

//go:embed lists/english.txt
var englishStopwordList string

//go:embed lists/spanish.txt
var spanishStopwordList string

//go:embed lists/french.txt
var frenchStopwordList string

//go:embed lists/italian.txt
var italianStopwordList string

//go:embed lists/russian.txt
var russianStopwordList string
