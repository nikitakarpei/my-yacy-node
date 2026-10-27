package featurehashing

import "strings"

type NgramSizes struct {
	Shortest int
	Longest  int
}

func CharacterNgramsFrom(document string, sizes NgramSizes) []string {
	var ngrams []string //nolint:prealloc // the amount depends on the length of each word
	for _, word := range strings.Fields(strings.ToLower(document)) {
		ngrams = append(ngrams, paddedWordNgramsFrom([]rune(" "+word+" "), sizes)...)
	}
	return ngrams
}

func paddedWordNgramsFrom(paddedWord []rune, sizes NgramSizes) []string {
	var ngrams []string
	for size := sizes.Shortest; size <= sizes.Longest; size++ {
		if size >= len(paddedWord) {
			return append(ngrams, string(paddedWord))
		}
		for start := 0; start+size <= len(paddedWord); start++ {
			ngrams = append(ngrams, string(paddedWord[start:start+size]))
		}
	}
	return ngrams
}

func WordNgramsFrom(document string, sizes NgramSizes) []string {
	words := strings.Fields(strings.ToLower(document))
	var ngrams []string
	for size := sizes.Shortest; size <= min(sizes.Longest, len(words)); size++ {
		for start := 0; start+size <= len(words); start++ {
			ngrams = append(ngrams, strings.Join(words[start:start+size], " "))
		}
	}
	return ngrams
}
