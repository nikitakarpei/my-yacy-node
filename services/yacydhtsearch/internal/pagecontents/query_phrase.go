package pagecontents

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type queryPhrase struct {
	firstWord  string
	secondWord string
}

type queryPhrases map[queryPhrase]struct{}

func queryPhrasesOf(queryWords []string) queryPhrases {
	phrases := queryPhrases{}
	for place := range len(queryWords) - 1 {
		phrases[queryPhrase{
			firstWord:  queryWords[place],
			secondWord: queryWords[place+1],
		}] = struct{}{}
		phrases[queryPhrase{
			firstWord:  queryWords[place+1],
			secondWord: queryWords[place],
		}] = struct{}{}
	}

	return phrases
}

func (phrases queryPhrases) hitsIn(text string) int {
	hits := 0
	wordBefore := ""
	for _, word := range yacymodel.PlacedWordsIn(text) {
		if _, askedFor := phrases[queryPhrase{firstWord: wordBefore, secondWord: word}]; askedFor {
			hits++
		}
		wordBefore = word
	}

	return hits
}
