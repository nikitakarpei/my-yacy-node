package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func findingsFrom(
	query searchquery.Query,
	discoveryRound discoveryRound,
	joinedDocuments distinctDocuments,
	answeredURLMetadataAsks []peerasks.AnsweredURLMetadataAsk,
) queryfindings.Findings {
	return queryfindings.Findings{
		QueryWords:    discoveryRound.queryWords,
		CompoundWords: query.CompoundWords,
		FoundDocuments: foundDocumentsFrom(
			discoveryRound, joinedDocuments, answeredURLMetadataAsks,
		),
		DocumentsHeldPerQueryWord: discoveryRound.
			amountOfDocumentsHeldPerQueryWord(),
	}
}

func foundDocumentsFrom(
	discoveryRound discoveryRound,
	joinedDocuments distinctDocuments,
	answeredURLMetadataAsks []peerasks.AnsweredURLMetadataAsk,
) []queryfindings.FoundDocument {
	documentsThePeersSent := queryfindings.EmptyDocumentsThePeersSent()
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
	for _, answeredAsk := range answeredURLMetadataAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			documentsThePeersSent.KeepMetadataThePeerSent(
				metadata, answeredAsk.Ask.Peer.Hash,
			)
		}
	}

	return documentsThePeersSent.FoundDocuments()
}
