// Package searchquery holds the words a client asked for outside the stopwords
// of their language, their compound words spelled as one, the word hashes that
// address them on the DHT ring, and the spelling a held ranking answers to.
package searchquery

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/stopwords"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Query struct {
	Words         []string
	CompoundWords []CompoundWord
	Exclusions    []string
	Language      string
}

func QueryFrom(raw, language string) Query {
	tokens := tokensOf(raw)
	var words, exclusions []string
	for _, token := range tokens {
		if token.excluded {
			exclusions = appendUnseen(exclusions, token.word)
			continue
		}
		words = appendUnseen(words, token.word)
	}
	contentWords := stopwords.ContentWordsOf(words, language)

	return Query{
		Words:         contentWords,
		CompoundWords: compoundWordsOf(tokens, contentWords),
		Exclusions:    exclusions,
		Language:      language,
	}
}

func (q Query) String() string {
	spelled := make([]string, 0, len(q.Words)+len(q.Exclusions)+1)
	spelled = append(spelled, q.Words...)
	for _, exclusion := range q.Exclusions {
		spelled = append(spelled, "-"+exclusion)
	}
	if q.Language != "" {
		spelled = append(spelled, "lr:"+q.Language)
	}

	return strings.Join(spelled, " ")
}

func (q Query) WordHashes() []yacymodel.Hash {
	return hashesOf(q.Words)
}

func (q Query) HashesOfWordsAndCompoundWordsUpTo(compoundWordsCeiling int) []yacymodel.Hash {
	hashes := q.WordHashes()
	for _, compound := range q.CompoundWords[:min(max(compoundWordsCeiling, 0), len(q.CompoundWords))] {
		hashes = append(hashes, compound.Hash)
	}

	return hashes
}

func (q Query) ExclusionHashes() []yacymodel.Hash {
	return hashesOf(q.Exclusions)
}

func hashesOf(words []string) []yacymodel.Hash {
	hashes := make([]yacymodel.Hash, 0, len(words))
	for _, word := range words {
		hashes = append(hashes, yacymodel.WordHash(word))
	}

	return hashes
}

type token struct {
	word     string
	excluded bool
}

func tokensOf(raw string) []token {
	var tokens []token
	for _, field := range strings.Fields(raw) {
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
