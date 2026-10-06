package peermatched

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

type settledAsks struct {
	query                          searchquery.Query
	growth                         queryfindings.Growth
	asks                           []wordpartitionasks.SettledAsk
	documentsThePeersSent          *queryfindings.DocumentsThePeersSent
	amountOfFoundDocumentsNotified int
}

func noSettledAsksYet(query searchquery.Query, growth queryfindings.Growth) *settledAsks {
	return &settledAsks{
		query:                 query,
		growth:                growth,
		documentsThePeersSent: queryfindings.EmptyDocumentsThePeersSent(),
	}
}

func (settled *settledAsks) keepAsTheySettle(
	settledAsksAsTheySettle <-chan wordpartitionasks.SettledAsk,
) {
	for settledAsk := range settledAsksAsTheySettle {
		settled.keep(settledAsk)
	}
}

func (settled *settledAsks) keep(settledAsk wordpartitionasks.SettledAsk) {
	settled.asks = append(settled.asks, settledAsk)
	for _, answer := range settledAsk.Answers {
		for _, listedDocument := range answer.ListedDocuments {
			metadata, sent := listedDocument.Metadata().Get()
			if !sent {
				continue
			}
			settled.documentsThePeersSent.KeepMetadataReplica(queryfindings.MetadataReplica{
				Holder: answer.Replica.Hash, Metadata: metadata,
			})
			if posting, sent := listedDocument.Posting().Get(); sent {
				settled.documentsThePeersSent.KeepPostingReplica(queryfindings.PostingReplica{
					Holder: answer.Replica.Hash, Word: settledAsk.Word, Posting: posting,
				})
			}
		}
	}
	settled.notifyIfFindingsGrew()
}

func (settled *settledAsks) notifyIfFindingsGrew() {
	findings := settled.findings()
	if len(findings.FoundDocuments) == settled.amountOfFoundDocumentsNotified {
		return
	}
	settled.amountOfFoundDocumentsNotified = len(findings.FoundDocuments)
	settled.growth.FindingsGrew(findings)
}

func (settled *settledAsks) findings() queryfindings.Findings {
	return queryfindings.Findings{
		QueryWords:     settled.query.WordHashes(),
		CompoundWords:  settled.query.CompoundWords,
		FoundDocuments: settled.documentsThePeersSent.FoundDocuments(),
	}
}

func (settled *settledAsks) amountOfPeersThatMatchedNothing() int {
	var peersThatMatchedNothing int
	for _, settledAsk := range settled.asks {
		for _, answer := range settledAsk.Answers {
			if slices.ContainsFunc(answer.ListedDocuments, isMatched) {
				continue
			}
			peersThatMatchedNothing++
		}
	}

	return peersThatMatchedNothing
}

func isMatched(listedDocument wordpartitionasks.ListedDocument) bool {
	return listedDocument.Metadata().Present()
}
