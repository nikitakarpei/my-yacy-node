package documentrelevance

import (
	"strings"
	"unicode"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type queryVocabulary struct {
	words         []yacymodel.Hash
	compoundWords []searchquery.CompoundWord
}

func queryVocabularyOf(findings queryfindings.Findings) queryVocabulary {
	return queryVocabulary{
		words:         findings.QueryWords,
		compoundWords: findings.CompoundWords,
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
		if _, held := wordsOfTheText[compoundWord.Hash()]; !held {
			continue
		}
		for _, word := range compoundWord.PartHashes() {
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
	for _, joinedWord := range wordsJoinedAtHyphensIn(text) {
		words[yacymodel.WordHash(joinedWord)] = struct{}{}
	}

	return words
}

const hyphen = '-'

func wordsJoinedAtHyphensIn(text string) []string {
	var joinedWords []string
	for hyphenatedWord := range strings.FieldsFuncSeq(text, isNeitherLetterNorDigitNorHyphen) {
		parts := strings.FieldsFunc(hyphenatedWord, isHyphen)
		for place := 1; place < len(parts); place++ {
			joinedWords = append(joinedWords, strings.ToLower(parts[place-1]+parts[place]))
		}
	}

	return joinedWords
}

func isNeitherLetterNorDigitNorHyphen(character rune) bool {
	return !unicode.IsLetter(character) && !unicode.IsDigit(character) && !isHyphen(character)
}

func isHyphen(character rune) bool {
	return character == hyphen
}
