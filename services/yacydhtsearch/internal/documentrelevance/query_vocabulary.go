package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type queryVocabulary struct {
	words         []yacymodel.Hash
	compoundWords []searchquery.CompoundWord
}

func queryVocabularyOf(answers queryanswers.AnsweredQuery) queryVocabulary {
	return queryVocabulary{
		words:         answers.QueryWords,
		compoundWords: answers.CompoundWords,
	}
}

func (vocabulary queryVocabulary) wordsIn(text string) []yacymodel.Hash {
	wordsOfTheText := hashesOfWordsIn(text)
	heldWords := vocabulary.wordsSpelledAsOneAmong(wordsOfTheText)
	for _, word := range vocabulary.words {
		if _, held := wordsOfTheText[word]; held {
			heldWords[word] = struct{}{}
		}
	}
	wordsInQueryOrder := make([]yacymodel.Hash, 0, len(heldWords))
	for _, word := range vocabulary.words {
		if _, held := heldWords[word]; held {
			wordsInQueryOrder = append(wordsInQueryOrder, word)
		}
	}

	return wordsInQueryOrder
}

func (vocabulary queryVocabulary) wordsSpelledAsOneAmong(
	wordsOfTheText map[yacymodel.Hash]struct{},
) map[yacymodel.Hash]struct{} {
	heldWords := make(map[yacymodel.Hash]struct{}, len(vocabulary.words))
	for _, compoundWord := range vocabulary.compoundWords {
		if _, held := wordsOfTheText[compoundWord.Hash]; !held {
			continue
		}
		for _, word := range compoundWord.WordHashes {
			heldWords[word] = struct{}{}
		}
	}

	return heldWords
}

func hashesOfWordsIn(text string) map[yacymodel.Hash]struct{} {
	spelledWords := yacymodel.WordsIn(text)
	words := make(map[yacymodel.Hash]struct{}, len(spelledWords))
	for _, spelledWord := range spelledWords {
		words[yacymodel.WordHash(spelledWord)] = struct{}{}
	}

	return words
}
