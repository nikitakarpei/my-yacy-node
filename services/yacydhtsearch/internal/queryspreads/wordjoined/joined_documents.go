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
	for document, amountOfWords := range amountOfQueryWordsPerDocument(answeredAsks) {
		if amountOfWords != len(queryWords) {
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
	amountOfWords := map[yacymodel.URLHash]int{}
	for _, answeredAsk := range answeredAsks {
		for _, document := range answeredAsk.DocumentsHeldForTheWord {
			ofDocument := queryWordOfDocument{document: document, queryWord: answeredAsk.Ask.Word}
			if _, counted := countedWords[ofDocument]; counted {
				continue
			}
			countedWords[ofDocument] = struct{}{}
			amountOfWords[document]++
		}
	}

	return amountOfWords
}

func joinedDocumentsWithoutMetadata(
	joinedDocuments map[yacymodel.URLHash]struct{},
	itemsInTheOrderOfEachAnswer [][]peeranswers.AnsweredItem,
) map[yacymodel.URLHash]struct{} {
	withoutMetadata := make(map[yacymodel.URLHash]struct{}, len(joinedDocuments))
	maps.Copy(withoutMetadata, joinedDocuments)
	for _, items := range itemsInTheOrderOfEachAnswer {
		for _, item := range items {
			delete(withoutMetadata, item.Metadata.Hash)
		}
	}

	return withoutMetadata
}
