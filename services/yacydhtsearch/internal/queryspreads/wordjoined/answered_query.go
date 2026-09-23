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
	for _, answeredAsk := range discoveryRound.answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if !joinedDocuments.contains(matchedDocument.Metadata.Hash) {
				continue
			}
			documentsThePeersSent.KeepDocumentThePeerMatched(
				answeredAsk.Ask.Peer.Hash,
				answeredAsk.Ask.Word,
				matchedDocument.Metadata,
				matchedDocument.Posting,
			)
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
