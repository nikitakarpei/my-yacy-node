package queryanswers

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type FoundDocument struct {
	Metadata        yacymodel.URLMetadata
	MatchedWords    map[yacymodel.Hash]WordCount
	QueryPhraseHits int
}

func (f FoundDocument) saturatedWith(text documenttext.DocumentText) FoundDocument {
	matchedWords := make(map[yacymodel.Hash]WordCount, len(text.HitsPerQueryWord))
	for word, hits := range text.HitsPerQueryWord {
		matchedWords[word] = WordCount{Hits: hits, TextWords: text.AmountOfWords}
	}
	f.MatchedWords = matchedWords
	f.QueryPhraseHits = text.QueryPhraseHits
	f.Metadata.Snippet = text.Snippet

	return f
}

func (f FoundDocument) CountedByAPeer() bool {
	for _, count := range f.MatchedWords {
		if count.CountedByAPeer() {
			return true
		}
	}

	return false
}
