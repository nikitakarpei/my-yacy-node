package queryreading

import (
	"github.com/nikitakarpei/yacy-rwi-node/stopwords"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func searchedWordsOf(words []string, language yacymodel.Language) []string {
	stopwordsOfTheQuery := stopwordsOf(words, language)
	wordsWithoutStopwords := make([]string, 0, len(words))
	for _, word := range words {
		if !stopwordsOfTheQuery.Contains(word) {
			wordsWithoutStopwords = append(wordsWithoutStopwords, word)
		}
	}
	if len(wordsWithoutStopwords) == 0 {
		return words
	}

	return wordsWithoutStopwords
}

func stopwordsOf(words []string, language yacymodel.Language) stopwords.Stopwords {
	stopwordsOfTheNamedLanguage, listed := stopwords.Of(language.String())
	if listed {
		return stopwordsOfTheNamedLanguage
	}
	languageCoveringMostWords, _ := stopwords.LanguageCoveringMostOf(words)
	stopwordsOfTheCoveringLanguage, _ := stopwords.Of(languageCoveringMostWords)

	return stopwordsOfTheCoveringLanguage
}
