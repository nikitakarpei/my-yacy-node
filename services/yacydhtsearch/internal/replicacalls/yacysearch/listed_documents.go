package yacysearch

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func listedDocumentsIn(
	answeredAsk peerasks.AnsweredSearchDocumentsAsk,
) []wordpartitionasks.ListedDocument {
	listed := listedDocumentsByHash{placeOfEachDocument: map[yacymodel.URLHash]int{}}
	for _, document := range answeredAsk.Abstract {
		listed.placeOf(document)
	}
	for _, matchedDocument := range answeredAsk.MatchedDocuments {
		listed.describe(matchedDocument)
	}

	return listed.documents
}

type listedDocumentsByHash struct {
	documents           []wordpartitionasks.ListedDocument
	placeOfEachDocument map[yacymodel.URLHash]int
}

func (listed *listedDocumentsByHash) placeOf(document yacymodel.URLHash) int {
	place, alreadyListed := listed.placeOfEachDocument[document]
	if alreadyListed {
		return place
	}
	place = len(listed.documents)
	listed.placeOfEachDocument[document] = place
	listed.documents = append(listed.documents, wordpartitionasks.ListedDocument{Hash: document})

	return place
}

func (listed *listedDocumentsByHash) describe(matchedDocument peerasks.MatchedDocument) {
	place := listed.placeOf(matchedDocument.Metadata.Hash)
	listed.documents[place].Metadata = yacymodel.Some(matchedDocument.Metadata)
	listed.documents[place].Posting = matchedDocument.Posting
}
