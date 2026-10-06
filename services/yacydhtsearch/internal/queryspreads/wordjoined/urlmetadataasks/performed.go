package urlmetadataasks

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Performed struct {
	AmountOfLookedUpDocuments             int
	AmountOfLookedUpDocumentsWithMetadata int
	EndReason                             EndReason
	AmountOfDocumentsCutOff               int
	AmountOfDocumentsPerAsk               []int
	TimeToFirstAsk                        yacymodel.Optional[time.Duration]
}

func PerformedFrom(urlMetadata Answers) Performed {
	lookedUpDocuments := lookedUpDocumentsAcross(urlMetadata.asks)

	return Performed{
		AmountOfLookedUpDocuments: len(lookedUpDocuments),
		AmountOfLookedUpDocumentsWithMetadata: amountOfDocumentsWithMetadataAmong(
			lookedUpDocuments, urlMetadata.answeredAsks,
		),
		EndReason:               urlMetadata.endReason,
		AmountOfDocumentsCutOff: urlMetadata.amountOfDocumentsCutOff,
		AmountOfDocumentsPerAsk: amountOfDocumentsPerAskIn(urlMetadata.asks),
		TimeToFirstAsk:          urlMetadata.timeToFirstAsk,
	}
}

func lookedUpDocumentsAcross(asks []peerasks.URLMetadataAsk) yacymodel.URLHashes {
	documents := yacymodel.URLHashes{}
	for _, ask := range asks {
		documents.AddEach(ask.Documents)
	}

	return documents
}

func amountOfDocumentsWithMetadataAmong(
	lookedUpDocuments yacymodel.URLHashes,
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) int {
	documentsWithMetadata := yacymodel.URLHashes{}
	for _, answeredAsk := range answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			if lookedUpDocuments.Contains(metadata.Hash) {
				documentsWithMetadata.Add(metadata.Hash)
			}
		}
	}

	return len(documentsWithMetadata)
}

func amountOfDocumentsPerAskIn(asks []peerasks.URLMetadataAsk) []int {
	amountOfDocumentsPerAsk := make([]int, 0, len(asks))
	for _, ask := range asks {
		amountOfDocumentsPerAsk = append(amountOfDocumentsPerAsk, len(ask.Documents))
	}

	return amountOfDocumentsPerAsk
}
