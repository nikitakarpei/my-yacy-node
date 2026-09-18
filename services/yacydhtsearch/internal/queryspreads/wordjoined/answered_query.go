package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
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
				foundDocuments = append(
					foundDocuments, queryanswers.FoundDocumentFrom(matchedDocument.Metadata),
				)
			}
			keepTheFirstCountOfTheWord(
				&foundDocuments[place],
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
			foundDocuments = append(foundDocuments, queryanswers.FoundDocumentFrom(metadata))
		}
	}

	return foundDocuments
}

func keepTheFirstCountOfTheWord(
	foundDocument *queryanswers.FoundDocument,
	word yacymodel.Hash,
	count peerasks.WordCount,
) {
	if !count.CountedByAPeer() {
		return
	}
	if _, alreadyCounted := foundDocument.HitsPerQueryWord[word]; alreadyCounted {
		return
	}
	foundDocument.HitsPerQueryWord[word] = count.Hits
	foundDocument.AmountOfWords = max(foundDocument.AmountOfWords, count.TextWords)
}
