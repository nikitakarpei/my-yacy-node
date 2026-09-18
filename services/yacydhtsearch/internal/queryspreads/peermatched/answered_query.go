package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
	queryWords []yacymodel.Hash,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:     queryWords,
		FoundDocuments: foundDocumentsFrom(answeredAsks, queryWords),
	}
}

func foundDocumentsFrom(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
	queryWords []yacymodel.Hash,
) []queryanswers.FoundDocument {
	countedWord, countedWordIsKnown := wordThePeersCountedFor(queryWords)

	var foundDocuments []queryanswers.FoundDocument
	placeOfEachDocument := map[yacymodel.URLHash]int{}
	for _, answeredAsk := range answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			place, alreadyFound := placeOfEachDocument[matchedDocument.Metadata.Hash]
			if !alreadyFound {
				place = len(foundDocuments)
				placeOfEachDocument[matchedDocument.Metadata.Hash] = place
				foundDocuments = append(foundDocuments, queryanswers.FoundDocument{
					Metadata:     matchedDocument.Metadata,
					MatchedWords: map[yacymodel.Hash]queryanswers.WordCount{},
				})
			}
			if !countedWordIsKnown {
				continue
			}
			keepTheFirstCountOfTheWord(
				foundDocuments[place].MatchedWords,
				countedWord,
				matchedDocument.CountOfAWordTheAskNamed,
			)
		}
	}

	return foundDocuments
}

func wordThePeersCountedFor(queryWords []yacymodel.Hash) (yacymodel.Hash, bool) {
	if len(queryWords) != 1 {
		return yacymodel.Hash{}, false
	}

	return queryWords[0], true
}

func keepTheFirstCountOfTheWord(
	matchedWords map[yacymodel.Hash]queryanswers.WordCount,
	word yacymodel.Hash,
	count queryanswers.WordCount,
) {
	if !count.CountedByAPeer() {
		return
	}
	if _, alreadyCounted := matchedWords[word]; alreadyCounted {
		return
	}
	matchedWords[word] = count
}
