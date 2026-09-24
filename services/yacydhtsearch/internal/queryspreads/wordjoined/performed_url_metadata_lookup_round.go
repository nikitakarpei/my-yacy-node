package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type PerformedURLMetadataLookupRound struct {
	AmountOfJoinedDocumentsWithMetadata   int
	AmountOfLookedUpDocuments             int
	AmountOfLookedUpDocumentsWithMetadata int
	End                                   URLMetadataLookupEnd
	AmountOfLookedUpDocumentsCutOff       int
}

func performedURLMetadataLookupRoundFrom(
	round urlMetadataLookupRound,
	joinedDocuments distinctDocuments,
) PerformedURLMetadataLookupRound {
	return PerformedURLMetadataLookupRound{
		AmountOfJoinedDocumentsWithMetadata: len(joinedDocuments) -
			len(round.documentsWithoutMetadata),
		AmountOfLookedUpDocuments: len(lookedUpDocumentsAcross(round.asks)),
		AmountOfLookedUpDocumentsWithMetadata: amountOfLookedUpDocumentsWithMetadata(
			round.asks, round.answeredAsks,
		),
		End:                             round.end,
		AmountOfLookedUpDocumentsCutOff: round.amountOfLookedUpDocumentsCutOff,
	}
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
		addLookedUpDocumentsOf(answeredAsk, lookedUpDocuments, documentsWithMetadata)
	}

	return len(documentsWithMetadata)
}

func addLookedUpDocumentsOf(
	answeredAsk peerasks.AnsweredURLMetadataAsk,
	lookedUpDocuments distinctDocuments,
	documentsWithMetadata distinctDocuments,
) {
	for _, metadata := range answeredAsk.MetadataOfEachDocument {
		if !lookedUpDocuments.contains(metadata.Hash) {
			continue
		}
		documentsWithMetadata.add(metadata.Hash)
	}
}
