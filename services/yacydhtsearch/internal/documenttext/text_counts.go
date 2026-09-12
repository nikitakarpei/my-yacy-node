package documenttext

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type textCounts struct {
	hitsPerQueryWord map[yacymodel.Hash]int
	queryPhraseHits  int
	amountOfWords    int
}

func textCountsOf(pageText string, queryWords []yacymodel.Hash) textCounts {
	counts := textCounts{hitsPerQueryWord: noHitsPerQueryWordOf(queryWords)}
	queryPhrases := queryPhrasesOf(queryWords)
	var wordBefore yacymodel.Hash
	for _, spelledWord := range yacymodel.PlacedWordsIn(pageText) {
		word := yacymodel.WordHash(spelledWord)
		counts.countTheWord(word)
		counts.countThePhrase(queryPhrases, wordBefore, word)
		wordBefore = word
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

func (c *textCounts) countThePhrase(
	queryPhrases map[queryPhrase]struct{},
	wordBefore yacymodel.Hash,
	word yacymodel.Hash,
) {
	phrase := queryPhrase{firstWord: wordBefore, secondWord: word}
	if _, askedFor := queryPhrases[phrase]; askedFor {
		c.queryPhraseHits++
	}
}
