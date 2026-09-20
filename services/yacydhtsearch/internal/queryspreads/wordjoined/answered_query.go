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
	return foundDocumentsCountedAcrossTheReplicas(
		distinctFoundDocumentsFrom(
			matchedAndHeldDocumentsRound, joinedDocuments, urlMetadataRound,
		),
		postingsPerDocumentAcrossReplicasFrom(matchedAndHeldDocumentsRound, joinedDocuments),
		matchedAndHeldDocumentsRound.placesGivenByPeersPerDocument(),
	)
}

func distinctFoundDocumentsFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinedDocuments distinctDocuments,
	urlMetadataRound urlMetadataRound,
) []queryanswers.FoundDocument {
	var foundDocuments []queryanswers.FoundDocument
	documentsAlreadyFound := distinctDocuments{}
	for _, answeredAsk := range matchedAndHeldDocumentsRound.answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if !joinedDocuments.contains(matchedDocument.Metadata.Hash) ||
				documentsAlreadyFound.contains(matchedDocument.Metadata.Hash) {
				continue
			}
			documentsAlreadyFound.add(matchedDocument.Metadata.Hash)
			foundDocuments = append(
				foundDocuments, queryanswers.FoundDocumentFrom(matchedDocument.Metadata),
			)
		}
	}
	for _, answeredAsk := range urlMetadataRound.answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			if documentsAlreadyFound.contains(metadata.Hash) {
				continue
			}
			documentsAlreadyFound.add(metadata.Hash)
			foundDocuments = append(foundDocuments, queryanswers.FoundDocumentFrom(metadata))
		}
	}

	return foundDocuments
}

func postingsPerDocumentAcrossReplicasFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinedDocuments distinctDocuments,
) map[yacymodel.URLHash]*queryanswers.PostingsOfOneDocumentAcrossReplicas {
	postingsPerDocument := map[yacymodel.URLHash]*queryanswers.PostingsOfOneDocumentAcrossReplicas{}
	for _, answeredAsk := range matchedAndHeldDocumentsRound.answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if !joinedDocuments.contains(matchedDocument.Metadata.Hash) {
				continue
			}
			if postingsPerDocument[matchedDocument.Metadata.Hash] == nil {
				postingsPerDocument[matchedDocument.Metadata.Hash] = &queryanswers.PostingsOfOneDocumentAcrossReplicas{}
			}
			postingsPerDocument[matchedDocument.Metadata.Hash].Take(
				answeredAsk.Ask.Word, matchedDocument.Posting,
			)
		}
	}

	return postingsPerDocument
}

func foundDocumentsCountedAcrossTheReplicas(
	foundDocuments []queryanswers.FoundDocument,
	postingsPerDocument map[yacymodel.URLHash]*queryanswers.PostingsOfOneDocumentAcrossReplicas,
	placesGivenByPeersPerDocument map[yacymodel.URLHash][]queryanswers.PlaceGivenByPeer,
) []queryanswers.FoundDocument {
	for place, foundDocument := range foundDocuments {
		foundDocuments[place].PlacesGivenByPeers = placesGivenByPeersPerDocument[foundDocument.Hash]
		postings, sentForTheDocument := postingsPerDocument[foundDocument.Hash]
		if !sentForTheDocument {
			continue
		}
		foundDocuments[place].HitsPerQueryWord = postings.HitsPerQueryWord()
		foundDocuments[place].AmountOfWords = postings.AmountOfWords()
		foundDocuments[place].LinkCounts = postings.LinkCounts()
	}

	return foundDocuments
}
