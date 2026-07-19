package documentsearch

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
func resultTopics(
	resources []yacymodel.URLMetadata,
	queryTerms []yacymodel.Hash,
) []string {
	excluded := termHashSet(queryTerms)
	frequency := make(map[string]int)
	for _, resource := range resources {
		for _, word := range titleWords(resource.Title) {
			if _, isQueryTerm := excluded[yacymodel.WordHash(word)]; isQueryTerm {
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
	words := fields[:0]
	for _, word := range fields {
		if len(word) >= minWordLength && onlyLetters(word) {
			words = append(words, word)
		}
	}

	return words
}

func onlyLetters(word string) bool {
	for _, r := range word {
		if r < 'a' || r > 'z' {
			return false
		}
	}

	return true
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

func termHashSet(terms []yacymodel.Hash) map[yacymodel.Hash]struct{} {
	if len(terms) == 0 {
		return nil
	}
	set := make(map[yacymodel.Hash]struct{}, len(terms))
	for _, term := range terms {
		set[term] = struct{}{}
	}

	return set
}
