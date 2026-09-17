package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func joinedDocumentsOf(
	anchor anchorOfTheQuery,
	answeredMatchedAndHeldDocumentsAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	answeredHeldDocumentsAsks []peerasks.AnsweredHeldDocumentsAsk,
	queryWords []yacymodel.Hash,
) map[yacymodel.URLHash]struct{} {
	documentsHeldPerQueryWord := documentsHeldPerQueryWordAcrossRoundsOf(
		answeredMatchedAndHeldDocumentsAsks, answeredHeldDocumentsAsks,
	)
	joinedDocuments := maps.Clone(anchor.documents)
	for _, queryWord := range queryWords {
		if queryWord == anchor.word {
			continue
		}
		joinedDocuments = documentsHeldForTheQueryWordAmong(
			joinedDocuments, documentsHeldPerQueryWord[queryWord],
		)
	}

	return joinedDocuments
}

func documentsHeldPerQueryWordAcrossRoundsOf(
	answeredMatchedAndHeldDocumentsAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	answeredHeldDocumentsAsks []peerasks.AnsweredHeldDocumentsAsk,
) map[yacymodel.Hash]map[yacymodel.URLHash]struct{} {
	documentsHeldPerQueryWord := map[yacymodel.Hash]map[yacymodel.URLHash]struct{}{}
	for _, answeredAsk := range answeredMatchedAndHeldDocumentsAsks {
		addDocumentsHeldForTheQueryWord(
			documentsHeldPerQueryWord, answeredAsk.Ask.Word, answeredAsk.DocumentsHeldForTheWord,
		)
	}
	for _, answeredAsk := range answeredHeldDocumentsAsks {
		addDocumentsHeldForTheQueryWord(
			documentsHeldPerQueryWord, answeredAsk.Ask.Word, answeredAsk.DocumentsHeldForTheWord,
		)
	}

	return documentsHeldPerQueryWord
}

func addDocumentsHeldForTheQueryWord(
	documentsHeldPerQueryWord map[yacymodel.Hash]map[yacymodel.URLHash]struct{},
	queryWord yacymodel.Hash,
	documentsHeldForTheWord []yacymodel.URLHash,
) {
	for _, document := range documentsHeldForTheWord {
		if documentsHeldPerQueryWord[queryWord] == nil {
			documentsHeldPerQueryWord[queryWord] = map[yacymodel.URLHash]struct{}{}
		}
		documentsHeldPerQueryWord[queryWord][document] = struct{}{}
	}
}

func documentsHeldForTheQueryWordAmong(
	documents map[yacymodel.URLHash]struct{},
	documentsHeldForTheQueryWord map[yacymodel.URLHash]struct{},
) map[yacymodel.URLHash]struct{} {
	keptDocuments := make(map[yacymodel.URLHash]struct{}, len(documents))
	for document := range documents {
		if _, held := documentsHeldForTheQueryWord[document]; !held {
			continue
		}
		keptDocuments[document] = struct{}{}
	}

	return keptDocuments
}

func joinedDocumentsWithoutMetadata(
	joinedDocuments map[yacymodel.URLHash]struct{},
	itemsInTheOrderOfEachPeerRanking [][]queryanswers.AnsweredItem,
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
