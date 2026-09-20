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
		FactsPerDocument: factsPerDocumentFrom(matchedAndHeldDocumentsRound, joinedDocuments),
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
	alreadyFoundDocuments := map[yacymodel.URLHash]struct{}{}
	for _, answeredAsk := range matchedAndHeldDocumentsRound.answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if !joinedDocuments.contains(matchedDocument.Metadata.Hash) {
				continue
			}
			if _, alreadyFound := alreadyFoundDocuments[matchedDocument.Metadata.Hash]; alreadyFound {
				continue
			}
			alreadyFoundDocuments[matchedDocument.Metadata.Hash] = struct{}{}
			foundDocuments = append(
				foundDocuments, queryanswers.FoundDocumentFrom(matchedDocument.Metadata),
			)
		}
	}
	for _, answeredAsk := range urlMetadataRound.answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			if _, alreadyFound := alreadyFoundDocuments[metadata.Hash]; alreadyFound {
				continue
			}
			alreadyFoundDocuments[metadata.Hash] = struct{}{}
			foundDocuments = append(foundDocuments, queryanswers.FoundDocumentFrom(metadata))
		}
	}

	return foundDocuments
}

func factsPerDocumentFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinedDocuments distinctDocuments,
) queryanswers.FactsPerDocument {
	factsPerDocument := queryanswers.FactsPerDocument{}
	for _, answeredAsk := range matchedAndHeldDocumentsRound.answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if !joinedDocuments.contains(matchedDocument.Metadata.Hash) {
				continue
			}
			factsPerDocument.KeepTheFirstPostingOfTheWord(
				matchedDocument.Metadata.Hash, answeredAsk.Ask.Word, matchedDocument.Posting,
			)
		}
	}

	return factsPerDocument
}
