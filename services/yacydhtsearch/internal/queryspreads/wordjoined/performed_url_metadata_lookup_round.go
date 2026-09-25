package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type PerformedURLMetadataLookupRound struct {
	AmountOfJoinedDocumentsWithMetadata   int
	AmountOfLookedUpDocuments             int
	AmountOfLookedUpDocumentsWithMetadata int
	EndReason                             URLMetadataLookupEndReason
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
		EndReason: round.endReason,
	}
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
