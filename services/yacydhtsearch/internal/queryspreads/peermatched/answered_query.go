package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

func answeredQueryFrom(
	settledAsks []wordpartitionasks.SettledAsk,
	query searchquery.Query,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:     query.WordHashes(),
		CompoundWords:  query.CompoundWords,
		FoundDocuments: foundDocumentsFrom(settledAsks),
	}
}

func foundDocumentsFrom(settledAsks []wordpartitionasks.SettledAsk) []queryanswers.FoundDocument {
	documentsThePeersSent := queryanswers.EmptyDocumentsThePeersSent()
	for _, settledAsk := range settledAsks {
		for _, answer := range settledAsk.Answers {
			for _, listedDocument := range answer.ListedDocuments {
				metadata, matched := listedDocument.Metadata.Get()
				if !matched {
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

	return documentsThePeersSent.FoundDocuments()
}
