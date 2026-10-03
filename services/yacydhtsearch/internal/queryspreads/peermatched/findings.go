package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

func findingsFrom(
	settledAsks []wordpartitionasks.SettledAsk,
	query searchquery.Query,
) queryfindings.Findings {
	return queryfindings.Findings{
		QueryWords:     query.WordHashes(),
		CompoundWords:  query.CompoundWords,
		FoundDocuments: foundDocumentsFrom(settledAsks),
	}
}

func foundDocumentsFrom(settledAsks []wordpartitionasks.SettledAsk) []queryfindings.FoundDocument {
	documentsThePeersSent := queryfindings.EmptyDocumentsThePeersSent()
	for _, settledAsk := range settledAsks {
		for _, answer := range settledAsk.Answers {
			for _, listedDocument := range answer.ListedDocuments {
				metadata, matched := listedDocument.Metadata.Get()
				if !matched {
					continue
				}
				documentsThePeersSent.KeepDocumentThePeerListed(
					answer.Replica.Hash,
					settledAsk.Word,
					metadata,
					listedDocument.Posting,
				)
			}
		}
	}

	return documentsThePeersSent.FoundDocuments()
}
