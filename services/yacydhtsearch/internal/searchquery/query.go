// Package searchquery holds the query a client asked for, the word hashes that
// address it on the DHT ring, and the spelling a held ranking answers to.
package searchquery

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Query struct {
	Terms      []string
	Exclusions []string
	Language   string
}

func QueryFrom(raw string) Query {
	var terms, exclusions []string
	for _, token := range tokensOf(raw) {
		if token.excluded {
			exclusions = appendUnseen(exclusions, token.word)
			continue
		}
		terms = appendUnseen(terms, token.word)
	}

	return Query{Terms: terms, Exclusions: exclusions}
}

func (q Query) String() string {
	spelled := make([]string, 0, len(q.Terms)+len(q.Exclusions)+1)
	spelled = append(spelled, q.Terms...)
	for _, exclusion := range q.Exclusions {
		spelled = append(spelled, "-"+exclusion)
	}
	if q.Language != "" {
		spelled = append(spelled, "lr:"+q.Language)
	}

	return strings.Join(spelled, " ")
}

func (q Query) TermHashes() []yacymodel.Hash {
	return hashesOf(q.Terms)
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
		word := strings.TrimLeft(field, "+-")
		word = strings.Trim(word, `"'`)
		word = strings.ToLower(strings.TrimSpace(word))
		if !yacymodel.WordIsIndexed(word) {
			continue
		}
		tokens = append(tokens, token{word: word, excluded: excluded})
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
