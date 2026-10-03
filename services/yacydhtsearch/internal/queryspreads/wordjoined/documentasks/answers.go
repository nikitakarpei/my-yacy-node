package documentasks

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Answers struct {
	SettledAsks []wordpartitionasks.SettledAsk
	Partitions  yacymodel.DHTRingPartitions
}

func (answers Answers) replicaAnswers() []wordpartitionasks.ReplicaAnswer {
	var replicaAnswers []wordpartitionasks.ReplicaAnswer
	for _, settledAsk := range answers.SettledAsks {
		replicaAnswers = append(replicaAnswers, settledAsk.Answers...)
	}

	return replicaAnswers
}

func (answers Answers) DocumentsWithoutMetadataAmong(
	documents yacymodel.URLHashes,
) yacymodel.URLHashes {
	documentsWithoutMetadata := maps.Clone(documents)
	for _, answer := range answers.replicaAnswers() {
		for _, listedDocument := range answer.ListedDocuments {
			if !listedDocument.Metadata.Present() {
				continue
			}
			delete(documentsWithoutMetadata, listedDocument.Hash)
		}
	}

	return documentsWithoutMetadata
}

func (answers Answers) DocumentHolders() documentholders.Holders {
	holders := documentholders.NoHolders()
	for _, settledAsk := range answers.SettledAsks {
		holders.AddHoldersIn(settledAsk.Answers)
	}

	return holders
}

type ListedDocumentWithMetadata struct {
	Replica  yacymodel.Hash
	Word     yacymodel.Hash
	Document yacymodel.URLHash
	Metadata yacymodel.URLMetadata
	Posting  yacymodel.Optional[yacymodel.RWIPosting]
}

func (answers Answers) ListedDocumentsWithMetadata() []ListedDocumentWithMetadata {
	var listedDocumentsWithMetadata []ListedDocumentWithMetadata
	for _, settledAsk := range answers.SettledAsks {
		for _, answer := range settledAsk.Answers {
			for _, listedDocument := range answer.ListedDocuments {
				metadata, sent := listedDocument.Metadata.Get()
				if !sent {
					continue
				}
				listedDocumentsWithMetadata = append(
					listedDocumentsWithMetadata,
					ListedDocumentWithMetadata{
						Replica:  answer.Replica.Hash,
						Word:     settledAsk.Word,
						Document: listedDocument.Hash,
						Metadata: metadata,
						Posting:  listedDocument.Posting,
					},
				)
			}
		}
	}

	return listedDocumentsWithMetadata
}
