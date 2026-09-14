package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func joinedDocumentsOf(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
	queryWords []yacymodel.Hash,
) map[yacymodel.URLHash]struct{} {
	joinedDocuments := map[yacymodel.URLHash]struct{}{}
	for document, amountOfQueryWords := range amountOfQueryWordsPerDocument(answeredAsks) {
		if amountOfQueryWords != len(queryWords) {
			continue
		}
		joinedDocuments[document] = struct{}{}
	}

	return joinedDocuments
}

type queryWordOfDocument struct {
	document  yacymodel.URLHash
	queryWord yacymodel.Hash
}

func amountOfQueryWordsPerDocument(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) map[yacymodel.URLHash]int {
	countedWords := map[queryWordOfDocument]struct{}{}
	amountOfQueryWordsPerDocument := map[yacymodel.URLHash]int{}
	for _, answeredAsk := range answeredAsks {
		for _, document := range answeredAsk.DocumentsHeldForTheWord {
			queryWordOfDocument := queryWordOfDocument{
				document: document, queryWord: answeredAsk.Ask.Word,
			}
			if _, counted := countedWords[queryWordOfDocument]; counted {
				continue
			}
			countedWords[queryWordOfDocument] = struct{}{}
			amountOfQueryWordsPerDocument[document]++
		}
	}

	return amountOfQueryWordsPerDocument
}

func joinedDocumentsWithoutMetadata(
	joinedDocuments map[yacymodel.URLHash]struct{},
	itemsInTheOrderOfEachPeerRanking [][]peeranswers.AnsweredItem,
) map[yacymodel.URLHash]struct{} {
	documentsWithoutMetadata := make(map[yacymodel.URLHash]struct{}, len(joinedDocuments))
	maps.Copy(documentsWithoutMetadata, joinedDocuments)
	for _, items := range itemsInTheOrderOfEachPeerRanking {
		for _, item := range items {
			delete(documentsWithoutMetadata, item.Metadata.Hash)
		}
	}

	return documentsWithoutMetadata
}
