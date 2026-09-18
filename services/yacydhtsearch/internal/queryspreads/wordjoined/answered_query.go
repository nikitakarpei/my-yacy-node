package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinedDocuments distinctDocuments,
	urlMetadataRound urlMetadataRound,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords: matchedAndHeldDocumentsRound.queryWords,
		FoundDocuments: foundDocumentsFrom(
			matchedAndHeldDocumentsRound, joinedDocuments, urlMetadataRound,
		),
		DocumentsHeldPerQueryWord: matchedAndHeldDocumentsRound.
			amountOfDocumentsHeldPerQueryWord(),
	}
}

func foundDocumentsFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinedDocuments distinctDocuments,
	urlMetadataRound urlMetadataRound,
) []queryanswers.FoundDocument {
	var foundDocuments []queryanswers.FoundDocument
	placeOfEachDocument := map[yacymodel.URLHash]int{}
	for _, answeredAsk := range matchedAndHeldDocumentsRound.answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if !joinedDocuments.contains(matchedDocument.Metadata.Hash) {
				continue
			}
			place, alreadyFound := placeOfEachDocument[matchedDocument.Metadata.Hash]
			if !alreadyFound {
				place = len(foundDocuments)
				placeOfEachDocument[matchedDocument.Metadata.Hash] = place
				foundDocuments = append(foundDocuments, queryanswers.FoundDocument{
					Metadata:     matchedDocument.Metadata,
					MatchedWords: map[yacymodel.Hash]queryanswers.WordCount{},
				})
			}
			keepTheFirstCountOfTheWord(
				foundDocuments[place].MatchedWords,
				answeredAsk.Ask.Word,
				matchedDocument.CountOfAWordTheAskNamed,
			)
		}
	}
	for _, answeredAsk := range urlMetadataRound.answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			if _, alreadyFound := placeOfEachDocument[metadata.Hash]; alreadyFound {
				continue
			}
			placeOfEachDocument[metadata.Hash] = len(foundDocuments)
			foundDocuments = append(foundDocuments, queryanswers.FoundDocument{Metadata: metadata})
		}
	}

	return foundDocuments
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
