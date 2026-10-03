package urlmetadataasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Performed struct {
	AmountOfAskedDocuments             int
	AmountOfAskedDocumentsWithMetadata int
	EndReason                          EndReason
	AmountOfDocumentsCutOff            int
	AmountOfDocumentsPerAsk            []int
}

func PerformedFrom(urlMetadata Answers) Performed {
	askedDocuments := askedDocumentsAcross(urlMetadata.asks)

	return Performed{
		AmountOfAskedDocuments: len(askedDocuments),
		AmountOfAskedDocumentsWithMetadata: amountOfDocumentsWithMetadataAmong(
			askedDocuments, urlMetadata.answeredAsks,
		),
		EndReason:               urlMetadata.endReason,
		AmountOfDocumentsCutOff: urlMetadata.amountOfDocumentsCutOff,
		AmountOfDocumentsPerAsk: amountOfDocumentsPerAskIn(urlMetadata.asks),
	}
}

func askedDocumentsAcross(asks []peerasks.URLMetadataAsk) yacymodel.URLHashes {
	documents := yacymodel.URLHashes{}
	for _, ask := range asks {
		documents.AddEach(ask.Documents)
	}

	return documents
}

func amountOfDocumentsWithMetadataAmong(
	askedDocuments yacymodel.URLHashes,
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) int {
	documentsWithMetadata := yacymodel.URLHashes{}
	for _, answeredAsk := range answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			if askedDocuments.Contains(metadata.Hash) {
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
