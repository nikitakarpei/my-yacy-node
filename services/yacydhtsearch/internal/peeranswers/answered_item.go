package peeranswers

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type AnsweredItem struct {
	Metadata        yacymodel.URLMetadata
	MatchedWords    map[yacymodel.Hash]WordCount
	QueryPhraseHits int
}

func (a AnsweredItem) MatchingTheWords(words []yacymodel.Hash) AnsweredItem {
	matchedWords := a.matchedWordsWithRoomFor(len(words))
	for _, word := range words {
		if _, matched := matchedWords[word]; matched {
			continue
		}
		matchedWords[word] = WordCount{}
	}
	a.MatchedWords = matchedWords

	return a
}

func (a AnsweredItem) carryingTheText(text documenttext.DocumentText) AnsweredItem {
	matchedWords := make(map[yacymodel.Hash]WordCount, len(a.MatchedWords))
	for word := range a.MatchedWords {
		matchedWords[word] = WordCount{
			Hits:      text.HitsPerQueryWord[word],
			TextWords: text.AmountOfWords,
		}
	}
	a.MatchedWords = matchedWords
	a.QueryPhraseHits = text.QueryPhraseHits
	a.Metadata.Snippet = text.Snippet

	return a
}

func (a AnsweredItem) CountedByAPeer() bool {
	for _, count := range a.MatchedWords {
		if count.CountedByAPeer() {
			return true
		}
	}

	return false
}

func (a AnsweredItem) withTheCountsOf(answeredAgain AnsweredItem) AnsweredItem {
	matchedWords := a.matchedWordsWithRoomFor(len(answeredAgain.MatchedWords))
	for word, count := range answeredAgain.MatchedWords {
		if held, matched := matchedWords[word]; matched && held.CountedByAPeer() {
			continue
		}
		matchedWords[word] = count
	}
	a.MatchedWords = matchedWords

	return a
}

func (a AnsweredItem) matchedWordsWithRoomFor(moreWords int) map[yacymodel.Hash]WordCount {
	matchedWords := make(map[yacymodel.Hash]WordCount, len(a.MatchedWords)+moreWords)
	maps.Copy(matchedWords, a.MatchedWords)

	return matchedWords
}
