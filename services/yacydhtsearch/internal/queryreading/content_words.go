package queryreading

import (
	"github.com/nikitakarpei/yacy-rwi-node/stopwords"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func contentWordsOf(words []string, language yacymodel.Language) []string {
	stopwordsOfTheQuery := stopwordsOfTheLanguageOf(words, language)
	contentWords := make([]string, 0, len(words))
	for _, word := range words {
		if stopwordsOfTheQuery.Contains(word) {
			continue
		}
		contentWords = append(contentWords, word)
	}
	if len(contentWords) == 0 {
		return words
	}

	return contentWords
}

func stopwordsOfTheLanguageOf(words []string, language yacymodel.Language) stopwords.Stopwords {
	stopwordsOfTheSpokenLanguage, listed := stopwords.Of(language.String())
	if listed {
		return stopwordsOfTheSpokenLanguage
	}
	languageCoveringMostWords, _ := stopwords.LanguageCoveringMostOf(words)
	stopwordsOfTheCoveringLanguage, _ := stopwords.Of(languageCoveringMostWords)

	return stopwordsOfTheCoveringLanguage
}
