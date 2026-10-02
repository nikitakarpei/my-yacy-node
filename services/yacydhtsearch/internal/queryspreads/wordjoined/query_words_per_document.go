package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type queryWordsPerDocument map[yacymodel.URLHash]map[yacymodel.Hash]struct{}

func (queryWordsOfEachDocument queryWordsPerDocument) addListingsIn(
	answers []wordpartitionasks.ReplicaAnswer,
	queryWords []yacymodel.Hash,
) {
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			if queryWordsOfEachDocument[listedDocument.Hash] == nil {
				queryWordsOfEachDocument[listedDocument.Hash] = map[yacymodel.Hash]struct{}{}
			}
			for _, queryWord := range queryWords {
				queryWordsOfEachDocument[listedDocument.Hash][queryWord] = struct{}{}
			}
		}
	}
}

func (queryWordsOfEachDocument queryWordsPerDocument) documentsWithEvery(
	queryWords []yacymodel.Hash,
) distinctDocuments {
	documentsWithEveryQueryWord := distinctDocuments{}
	for document, queryWordsOfTheDocument := range queryWordsOfEachDocument {
		if !containsEvery(queryWordsOfTheDocument, queryWords) {
			continue
		}
		documentsWithEveryQueryWord.add(document)
	}

	return documentsWithEveryQueryWord
}

func containsEvery(
	queryWordsOfTheDocument map[yacymodel.Hash]struct{},
	queryWords []yacymodel.Hash,
) bool {
	for _, queryWord := range queryWords {
		if _, listed := queryWordsOfTheDocument[queryWord]; !listed {
			return false
		}
	}

	return true
}
