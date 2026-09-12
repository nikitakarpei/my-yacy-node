// Package stopwords holds the Snowball stopword lists of English, German,
// French, Spanish, Italian and Russian, and picks the words that carry the
// subject of a query out of the words the client spoke.
package stopwords

import (
	_ "embed"
	"strings"
)

func ContentWordsOf(words []string, language string) []string {
	stopwordsOfTheQuery := stopwordsOfTheLanguageOf(words, language)
	contentWords := make([]string, 0, len(words))
	for _, word := range words {
		if _, stopword := stopwordsOfTheQuery[word]; stopword {
			continue
		}
		contentWords = append(contentWords, word)
	}
	if len(contentWords) == 0 {
		return words
	}

	return contentWords
}

func stopwordsOfTheLanguageOf(words []string, language string) map[string]struct{} {
	stopwordsOfTheSpokenLanguage, listed := stopwordsPerLanguage[languageCodeIn(language)]
	if listed {
		return stopwordsOfTheSpokenLanguage
	}

	return stopwordsPerLanguage[theOneLanguageCoveringMostWordsIn(words)]
}

func languageCodeIn(language string) string {
	return strings.TrimPrefix(strings.ToLower(language), "lang_")
}

func theOneLanguageCoveringMostWordsIn(words []string) string {
	languageOfTheMostCoveredWords := ""
	mostCoveredWords := 0
	amountOfLanguagesCoveringAsMany := 0
	for language, stopwordsOfTheLanguage := range stopwordsPerLanguage {
		coveredWords := amountOfCoveredWordsIn(words, stopwordsOfTheLanguage)
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
		return ""
	}

	return languageOfTheMostCoveredWords
}

func amountOfCoveredWordsIn(words []string, stopwordsOfTheLanguage map[string]struct{}) int {
	amountOfCoveredWords := 0
	for _, word := range words {
		if _, covered := stopwordsOfTheLanguage[word]; covered {
			amountOfCoveredWords++
		}
	}

	return amountOfCoveredWords
}

var stopwordsPerLanguage = map[string]map[string]struct{}{
	"de": stopwordsInTheList(germanStopwordList),
	"en": stopwordsInTheList(englishStopwordList),
	"es": stopwordsInTheList(spanishStopwordList),
	"fr": stopwordsInTheList(frenchStopwordList),
	"it": stopwordsInTheList(italianStopwordList),
	"ru": stopwordsInTheList(russianStopwordList),
}

func stopwordsInTheList(list string) map[string]struct{} {
	stopwordsOfTheLanguage := make(map[string]struct{})
	for _, word := range strings.Fields(list) {
		stopwordsOfTheLanguage[word] = struct{}{}
	}

	return stopwordsOfTheLanguage
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
