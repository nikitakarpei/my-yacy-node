package pagecontents

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type textCounts struct {
	hitsPerQueryWord map[yacymodel.Hash]int
	amountOfWords    int
}

func textCountsOf(pageText string, queryWords []yacymodel.Hash) textCounts {
	counts := textCounts{hitsPerQueryWord: noHitsPerQueryWordOf(queryWords)}
	for _, spelledWord := range yacymodel.PlacedWordsIn(pageText) {
		counts.countTheWord(yacymodel.WordHash(spelledWord))
	}

	return counts
}

func noHitsPerQueryWordOf(queryWords []yacymodel.Hash) map[yacymodel.Hash]int {
	hitsPerQueryWord := make(map[yacymodel.Hash]int, len(queryWords))
	for _, queryWord := range queryWords {
		hitsPerQueryWord[queryWord] = 0
	}

	return hitsPerQueryWord
}

func (c *textCounts) countTheWord(word yacymodel.Hash) {
	c.amountOfWords++
	if _, askedFor := c.hitsPerQueryWord[word]; askedFor {
		c.hitsPerQueryWord[word]++
	}
}
