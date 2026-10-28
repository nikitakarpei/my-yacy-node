package wordjoined

import (
	"slices"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type documentsTheSpreadFound struct {
	mutex                          sync.Mutex
	query                          searchquery.Query
	measurement                    *documentamounts.Measurement
	growth                         queryfindings.Growth
	documentsThePeersSent          *queryfindings.DocumentsThePeersSent
	documentsWithMetadata          yacymodel.URLHashes
	joinedDocuments                yacymodel.URLHashes
	amountOfFoundDocumentsNotified int
}

func noDocumentsFoundYet(
	query searchquery.Query,
	measurement *documentamounts.Measurement,
	growth queryfindings.Growth,
) *documentsTheSpreadFound {
	return &documentsTheSpreadFound{
		query:                 query,
		measurement:           measurement,
		growth:                growth,
		documentsThePeersSent: queryfindings.EmptyDocumentsThePeersSent(),
	}
}

func (found *documentsTheSpreadFound) WordPartitionAnswered(
	word yacymodel.Hash,
	_ uint,
	answers []wordpartitionasks.ReplicaAnswer,
) {
	found.mutex.Lock()
	defer found.mutex.Unlock()
	var documentsWithMetadata yacymodel.URLHashesBuilder
	for _, answer := range answers {
		for _, matchedDocument := range answer.MatchedDocuments {
			found.documentsThePeersSent.KeepMetadataReplica(queryfindings.MetadataReplica{
				Holder: answer.Holder.Hash, Metadata: matchedDocument.Metadata,
			})
			documentsWithMetadata.Add(matchedDocument.Metadata.Hash)
			if posting, sent := matchedDocument.Posting.Get(); sent {
				found.documentsThePeersSent.KeepPostingReplica(queryfindings.PostingReplica{
					Holder: answer.Holder.Hash, Word: word, Posting: posting,
				})
			}
		}
	}
	found.keepDocumentsWithMetadata(documentsWithMetadata.URLHashes())
	found.notifyIfFindingsGrew()
}

func (found *documentsTheSpreadFound) keepDocumentsWithMetadata(documents yacymodel.URLHashes) {
	found.documentsWithMetadata = yacymodel.UnionOf(found.documentsWithMetadata, documents)
}

func (found *documentsTheSpreadFound) notifyIfFindingsGrew() {
	amountOfFoundDocuments := found.amountOfFoundDocuments()
	if amountOfFoundDocuments == found.amountOfFoundDocumentsNotified {
		return
	}
	found.amountOfFoundDocumentsNotified = amountOfFoundDocuments
	found.growth.FindingsGrew(found.findingsSoFar())
}

func (found *documentsTheSpreadFound) amountOfFoundDocuments() int {
	amount := 0
	for document := range found.joinedDocuments.All() {
		if found.documentsWithMetadata.Contains(document) {
			amount++
		}
	}

	return amount
}

func (found *documentsTheSpreadFound) findingsSoFar() queryfindings.Findings {
	return queryfindings.Findings{
		QueryWords:                found.query.WordHashes(),
		CompoundWords:             found.query.CompoundWords,
		FoundDocuments:            found.joinedAmong(found.documentsThePeersSent.FoundDocuments()),
		DocumentsHeldPerQueryWord: found.measurement.HeldPerQueryWord(),
	}
}

func (found *documentsTheSpreadFound) joinedAmong(
	foundDocuments []queryfindings.FoundDocument,
) []queryfindings.FoundDocument {
	return slices.DeleteFunc(foundDocuments, func(foundDocument queryfindings.FoundDocument) bool {
		return !found.joinedDocuments.Contains(foundDocument.Hash)
	})
}

func (found *documentsTheSpreadFound) PeerSentURLMetadata(
	peer yacymodel.Hash,
	metadataOfEachDocument []yacymodel.URLMetadata,
) {
	found.mutex.Lock()
	defer found.mutex.Unlock()
	var documentsWithMetadata yacymodel.URLHashesBuilder
	for _, metadata := range metadataOfEachDocument {
		found.documentsThePeersSent.KeepMetadataReplica(queryfindings.MetadataReplica{
			Holder: peer, Metadata: metadata,
		})
		documentsWithMetadata.Add(metadata.Hash)
	}
	found.keepDocumentsWithMetadata(documentsWithMetadata.URLHashes())
	found.notifyIfFindingsGrew()
}

func (found *documentsTheSpreadFound) DocumentsJoined(documents yacymodel.URLHashes) {
	found.mutex.Lock()
	defer found.mutex.Unlock()
	found.joinedDocuments = yacymodel.UnionOf(found.joinedDocuments, documents)
	found.notifyIfFindingsGrew()
}

func (found *documentsTheSpreadFound) withoutMetadataAmong(
	documents yacymodel.URLHashes,
) yacymodel.URLHashes {
	found.mutex.Lock()
	defer found.mutex.Unlock()

	return documents.Without(found.documentsWithMetadata)
}

func (found *documentsTheSpreadFound) findings() queryfindings.Findings {
	found.mutex.Lock()
	defer found.mutex.Unlock()

	return found.findingsSoFar()
}
