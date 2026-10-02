package urlmetadataasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Answers struct {
	asks                    []peerasks.URLMetadataAsk
	answeredAsks            []peerasks.AnsweredURLMetadataAsk
	endReason               EndReason
	amountOfDocumentsCutOff int
}

type EndReason string

const (
	EndedByCoverage        EndReason = "coverage"
	EndedByEveryAskSettled EndReason = "every ask settled"
	EndedByCutoff          EndReason = "cut off"
)

type SentMetadata struct {
	Peer     yacymodel.Hash
	Metadata yacymodel.URLMetadata
}

func (answers Answers) MetadataThePeersSent() []SentMetadata {
	var sentMetadata []SentMetadata
	for _, answeredAsk := range answers.answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			sentMetadata = append(sentMetadata, SentMetadata{
				Peer:     answeredAsk.Ask.Peer.Hash,
				Metadata: metadata,
			})
		}
	}

	return sentMetadata
}
