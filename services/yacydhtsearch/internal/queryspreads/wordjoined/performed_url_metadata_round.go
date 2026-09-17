package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type PerformedURLMetadataRound struct {
	AmountOfJoinedDocumentsWithMetadata   int
	AmountOfLookedUpDocuments             int
	AmountOfLookedUpDocumentsWithMetadata int
}

func performedURLMetadataRoundFrom(
	round urlMetadataRound,
	joinedDocuments distinctDocuments,
) PerformedURLMetadataRound {
	return PerformedURLMetadataRound{
		AmountOfJoinedDocumentsWithMetadata: len(joinedDocuments) -
			len(round.documentsWithoutMetadata),
		AmountOfLookedUpDocuments: len(lookedUpDocumentsAcross(round.asks)),
		AmountOfLookedUpDocumentsWithMetadata: amountOfLookedUpDocumentsWithMetadata(
			round.asks, round.answeredAsks,
		),
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
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			if !lookedUpDocuments.contains(metadata.Hash) {
				continue
			}
			documentsWithMetadata.add(metadata.Hash)
		}
	}

	return len(documentsWithMetadata)
}
