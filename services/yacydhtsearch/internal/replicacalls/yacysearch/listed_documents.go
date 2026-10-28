package yacysearch

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func listedDocumentsIn(answeredAsk peerasks.AnsweredSearchDocumentsAsk) yacymodel.URLHashes {
	var matchedDocuments yacymodel.URLHashesBuilder
	for _, matchedDocument := range answeredAsk.MatchedDocuments {
		matchedDocuments.Add(matchedDocument.Metadata.Hash)
	}

	return yacymodel.UnionOf(answeredAsk.Abstract, matchedDocuments.URLHashes())
}

func matchedDocumentsIn(
	answeredAsk peerasks.AnsweredSearchDocumentsAsk,
) []wordpartitionasks.MatchedDocument {
	placeOfEachDocument := make(map[yacymodel.URLHash]int, len(answeredAsk.MatchedDocuments))
	matchedDocuments := make(
		[]wordpartitionasks.MatchedDocument,
		0,
		len(answeredAsk.MatchedDocuments),
	)
	for _, matchedDocument := range answeredAsk.MatchedDocuments {
		document := wordpartitionasks.MatchedDocument{
			Metadata: matchedDocument.Metadata,
			Posting:  matchedDocument.Posting,
		}
		if place, matchedBefore := placeOfEachDocument[matchedDocument.Metadata.Hash]; matchedBefore {
			matchedDocuments[place] = document

			continue
		}
		placeOfEachDocument[matchedDocument.Metadata.Hash] = len(matchedDocuments)
		matchedDocuments = append(matchedDocuments, document)
	}

	return matchedDocuments
}
