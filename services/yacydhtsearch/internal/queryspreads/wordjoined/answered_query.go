package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func answeredQueryFrom(
	query searchquery.Query,
	discoveryRound discoveryRound,
	joinedDocuments distinctDocuments,
	urlMetadataLookupRound urlMetadataLookupRound,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:    discoveryRound.queryWords,
		CompoundWords: query.CompoundWords,
		FoundDocuments: foundDocumentsFrom(
			discoveryRound, joinedDocuments, urlMetadataLookupRound,
		),
		DocumentsHeldPerQueryWord: discoveryRound.
			amountOfDocumentsHeldPerQueryWord(),
	}
}

func foundDocumentsFrom(
	discoveryRound discoveryRound,
	joinedDocuments distinctDocuments,
	urlMetadataLookupRound urlMetadataLookupRound,
) []queryanswers.FoundDocument {
	documentsThePeersSent := queryanswers.EmptyDocumentsThePeersSent()
	for _, settledAsk := range discoveryRound.settledAsks {
		for _, answer := range settledAsk.Answers {
			for _, listedDocument := range answer.ListedDocuments {
				metadata, matched := listedDocument.Metadata.Get()
				if !matched || !joinedDocuments.contains(listedDocument.Hash) {
					continue
				}
				documentsThePeersSent.KeepDocumentThePeerMatched(
					answer.Replica.Hash,
					settledAsk.Word,
					metadata,
					listedDocument.Posting,
				)
			}
		}
	}
	for _, answeredAsk := range urlMetadataLookupRound.answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			documentsThePeersSent.KeepMetadataThePeerSent(
				metadata, answeredAsk.Ask.Peer.Hash,
			)
		}
	}

	return documentsThePeersSent.FoundDocuments()
}
