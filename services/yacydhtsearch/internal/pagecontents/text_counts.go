package pagecontents

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type textCounts struct {
	hitsPerQueryWord map[string]int
	amountOfWords    int
}

func textCountsOf(pageText string, queryWords []string) textCounts {
	counts := textCounts{hitsPerQueryWord: noHitsPerQueryWordOf(queryWords)}
	for _, word := range yacymodel.PlacedWordsIn(pageText) {
		counts.countTheWord(word)
	}

	return counts
}

func noHitsPerQueryWordOf(queryWords []string) map[string]int {
	hitsPerQueryWord := make(map[string]int, len(queryWords))
	for _, queryWord := range queryWords {
		hitsPerQueryWord[queryWord] = 0
	}

	return hitsPerQueryWord
}

func (c *textCounts) countTheWord(word string) {
	c.amountOfWords++
	if _, askedFor := c.hitsPerQueryWord[word]; askedFor {
		c.hitsPerQueryWord[word]++
	}
}

func (c textCounts) hitsPerWordHash() map[yacymodel.Hash]int {
	hitsPerWordHash := make(map[yacymodel.Hash]int, len(c.hitsPerQueryWord))
	for queryWord, hits := range c.hitsPerQueryWord {
		hitsPerWordHash[yacymodel.WordHash(queryWord)] = hits
	}

	return hitsPerWordHash
}
