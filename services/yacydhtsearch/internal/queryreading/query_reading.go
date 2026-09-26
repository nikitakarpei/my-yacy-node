// Package queryreading reads the text and the language a client sent into the
// query the search asks: each word once, the exclusions apart, the stopwords of
// the language left out, and the compound words its neighbouring words form.
package queryreading

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/stopwords"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func QueryFrom(text, language string) searchquery.Query {
	tokens := tokensOf(text)
	var words, exclusions []string
	for _, token := range tokens {
		if token.excluded {
			exclusions = appendUnseen(exclusions, token.word)
			continue
		}
		words = appendUnseen(words, token.word)
	}
	contentWords := stopwords.ContentWordsOf(words, language)

	return searchquery.Query{
		Words:         contentWords,
		CompoundWords: compoundWordsOf(tokens, contentWords),
		Exclusions:    exclusions,
		Language:      language,
	}
}

type token struct {
	word     string
	excluded bool
}

func tokensOf(text string) []token {
	var tokens []token
	for _, field := range strings.Fields(text) {
		excluded := strings.HasPrefix(field, "-")
		for _, word := range yacymodel.WordsIn(field) {
			tokens = append(tokens, token{word: word, excluded: excluded})
		}
	}

	return tokens
}

func appendUnseen(words []string, word string) []string {
	for _, seen := range words {
		if seen == word {
			return words
		}
	}

	return append(words, word)
}
