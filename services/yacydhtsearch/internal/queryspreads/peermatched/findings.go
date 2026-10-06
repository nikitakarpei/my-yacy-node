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
				documentsThePeersSent.KeepMetadataReplica(queryfindings.MetadataReplica{
					Holder: answer.Replica.Hash, Metadata: metadata,
				})
				if posting, sent := listedDocument.Posting.Get(); sent {
					documentsThePeersSent.KeepPostingReplica(queryfindings.PostingReplica{
						Holder: answer.Replica.Hash, Word: settledAsk.Word, Posting: posting,
					})
				}
			}
		}
	}

	return documentsThePeersSent.FoundDocuments()
}
