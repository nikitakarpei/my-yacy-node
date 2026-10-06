package wordjoined

import (
	"slices"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type documentsTheSpreadFound struct {
	mutex                 *sync.Mutex
	documentsThePeersSent *queryfindings.DocumentsThePeersSent
}

func noDocumentsFoundYet() documentsTheSpreadFound {
	return documentsTheSpreadFound{
		mutex:                 &sync.Mutex{},
		documentsThePeersSent: queryfindings.EmptyDocumentsThePeersSent(),
	}
}

func (found documentsTheSpreadFound) WordPartitionAnswered(
	word yacymodel.Hash,
	_ uint,
	answers []wordpartitionasks.ReplicaAnswer,
) {
	found.mutex.Lock()
	defer found.mutex.Unlock()
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			metadata, sent := listedDocument.Metadata.Get()
			if !sent {
				continue
			}
			found.documentsThePeersSent.KeepMetadataReplica(queryfindings.MetadataReplica{
				Holder: answer.Replica.Hash, Metadata: metadata,
			})
			if posting, sent := listedDocument.Posting.Get(); sent {
				found.documentsThePeersSent.KeepPostingReplica(queryfindings.PostingReplica{
					Holder: answer.Replica.Hash, Word: word, Posting: posting,
				})
			}
		}
	}
}

func (found documentsTheSpreadFound) PeerSentURLMetadata(
	peer yacymodel.Hash,
	metadataOfEachDocument []yacymodel.URLMetadata,
) {
	found.mutex.Lock()
	defer found.mutex.Unlock()
	for _, metadata := range metadataOfEachDocument {
		found.documentsThePeersSent.KeepMetadataReplica(queryfindings.MetadataReplica{
			Holder: peer, Metadata: metadata,
		})
	}
}

func (found documentsTheSpreadFound) findingsFor(
	query searchquery.Query,
	joinedDocuments yacymodel.URLHashes,
	documentsHeldPerQueryWord map[yacymodel.Hash]int,
) queryfindings.Findings {
	return queryfindings.Findings{
		QueryWords:                query.WordHashes(),
		CompoundWords:             query.CompoundWords,
		FoundDocuments:            found.among(joinedDocuments),
		DocumentsHeldPerQueryWord: documentsHeldPerQueryWord,
	}
}

func (found documentsTheSpreadFound) among(
	joinedDocuments yacymodel.URLHashes,
) []queryfindings.FoundDocument {
	found.mutex.Lock()
	defer found.mutex.Unlock()

	return slices.DeleteFunc(
		found.documentsThePeersSent.FoundDocuments(),
		func(foundDocument queryfindings.FoundDocument) bool {
			return !joinedDocuments.Contains(foundDocument.Hash)
		},
	)
}
