// Package titletopics names the words that recur in the titles of the returned
// documents and are not query terms.
package titletopics

import (
	"sort"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	maxTopics     = 5
	minWordLength = 3
)

const titleWordSeparators = " /()-:_.,?!'\""

// Topics come only from the returned page, not every matched document as YaCy does,
// to keep search latency bounded; the field is a navigation hint, so the narrower
// sample is acceptable.
func TopicsFromTitles(
	documentMetadata []yacymodel.URLMetadata,
	queryTerms []yacymodel.Hash,
) []string {
	queryTermHashes := termSet(queryTerms)
	frequency := make(map[string]int)
	for _, metadata := range documentMetadata {
		for _, word := range titleWords(metadata.Title) {
			if _, isQueryTerm := queryTermHashes[yacymodel.WordHash(word)]; isQueryTerm {
				continue
			}
			if isUnhelpfulTopicWord(word) {
				continue
			}
			frequency[word]++
		}
	}

	return mostFrequentTopics(frequency)
}

func titleWords(title string) []string {
	fields := strings.FieldsFunc(strings.ToLower(title), func(r rune) bool {
		return strings.ContainsRune(titleWordSeparators, r)
	})
	words := make([]string, 0, len(fields))
	for _, word := range fields {
		if len(word) >= minWordLength && containsOnlyLetters(word) {
			words = append(words, word)
		}
	}

	return words
}

func containsOnlyLetters(word string) bool {
	for _, r := range word {
		if r < 'a' || r > 'z' {
			return false
		}
	}

	return true
}

// Deliberate divergence from YaCy: only its hardcoded common-word blocklist is
// applied, not the larger stopword and badword sets.
var unhelpfulTopicWords = map[string]struct{}{
	"http": {}, "html": {}, "php": {}, "ftp": {}, "www": {}, "com": {},
	"org": {}, "net": {}, "gov": {}, "edu": {}, "index": {}, "home": {},
	"page": {}, "for": {}, "usage": {}, "the": {}, "and": {}, "zum": {},
	"der": {}, "die": {}, "das": {}, "und": {}, "zur": {}, "bzw": {},
	"mit": {}, "blog": {}, "wiki": {}, "aus": {}, "bei": {}, "off": {},
}

func isUnhelpfulTopicWord(word string) bool {
	_, found := unhelpfulTopicWords[word]

	return found
}

func mostFrequentTopics(frequency map[string]int) []string {
	words := make([]string, 0, len(frequency))
	for word := range frequency {
		words = append(words, word)
	}
	sort.Slice(words, func(i, j int) bool {
		if frequency[words[i]] != frequency[words[j]] {
			return frequency[words[i]] > frequency[words[j]]
		}

		return words[i] < words[j]
	})
	if len(words) > maxTopics {
		words = words[:maxTopics]
	}

	return words
}

func termSet(terms []yacymodel.Hash) map[yacymodel.Hash]struct{} {
	if len(terms) == 0 {
		return nil
	}
	set := make(map[yacymodel.Hash]struct{}, len(terms))
	for _, term := range terms {
		set[term] = struct{}{}
	}

	return set
}
