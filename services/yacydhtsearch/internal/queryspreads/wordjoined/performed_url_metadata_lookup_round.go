package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedURLMetadataLookupRound struct {
	AmountOfLookedUpDocuments             int
	AmountOfLookedUpDocumentsWithMetadata int
	EndReason                             URLMetadataLookupEndReason
	AmountOfDocumentsCutOffDuringLookup   int
	AmountOfDocumentsPerAsk               []int
}

func performedURLMetadataLookupRoundFrom(
	round urlMetadataLookupRound,
) PerformedURLMetadataLookupRound {
	return PerformedURLMetadataLookupRound{
		AmountOfLookedUpDocuments: len(lookedUpDocumentsAcross(round.asks)),
		AmountOfLookedUpDocumentsWithMetadata: amountOfLookedUpDocumentsWithMetadata(
			round.asks, round.answeredAsks,
		),
		EndReason:                           round.endReason,
		AmountOfDocumentsCutOffDuringLookup: round.amountOfDocumentsCutOffDuringLookup,
		AmountOfDocumentsPerAsk:             amountOfDocumentsPerAskIn(round.asks),
	}
}

func amountOfDocumentsPerAskIn(asks []peerasks.URLMetadataAsk) []int {
	amountOfDocumentsPerAsk := make([]int, 0, len(asks))
	for _, ask := range asks {
		amountOfDocumentsPerAsk = append(amountOfDocumentsPerAsk, len(ask.Documents))
	}

	return amountOfDocumentsPerAsk
}

func lookedUpDocumentsAcross(
	asks []peerasks.URLMetadataAsk,
) distinctDocuments {
	documents := distinctDocuments{}
	for _, ask := range asks {
		for _, document := range ask.Documents {
			documents.add(document)
		}
	}

	return documents
}

func amountOfLookedUpDocumentsWithMetadata(
	asks []peerasks.URLMetadataAsk,
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) int {
	lookedUpDocuments := lookedUpDocumentsAcross(asks)
	documentsWithMetadata := distinctDocuments{}
	for _, answeredAsk := range answeredAsks {
		documentsWithMetadata.addEach(
			lookedUpDocumentsWithMetadataIn(answeredAsk, lookedUpDocuments),
		)
	}

	return len(documentsWithMetadata)
}

func lookedUpDocumentsWithMetadataIn(
	answeredAsk peerasks.AnsweredURLMetadataAsk,
	lookedUpDocuments distinctDocuments,
) []yacymodel.URLHash {
	var documentsWithMetadata []yacymodel.URLHash
	for _, metadata := range answeredAsk.MetadataOfEachDocument {
		if lookedUpDocuments.contains(metadata.Hash) {
			documentsWithMetadata = append(documentsWithMetadata, metadata.Hash)
		}
	}

	return documentsWithMetadata
}
