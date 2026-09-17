package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedURLMetadataRound struct {
	AmountOfJoinedDocumentsWithMetadata int
	AmountOfDocumentsAskedMetadataFor   int
	AmountOfAskedDocumentsWithMetadata  int
}

func performedURLMetadataRoundFrom(
	round urlMetadataRound,
	joinOfTheQuery joinOfTheQuery,
) PerformedURLMetadataRound {
	return PerformedURLMetadataRound{
		AmountOfJoinedDocumentsWithMetadata: len(joinOfTheQuery.joinedDocuments) -
			len(round.documentsWithoutMetadata),
		AmountOfDocumentsAskedMetadataFor: len(documentsAcrossURLMetadataAsks(round.asks)),
		AmountOfAskedDocumentsWithMetadata: amountOfAskedDocumentsWithMetadata(
			round.asks, round.answeredAsks,
		),
	}
}

func documentsAcrossURLMetadataAsks(
	asks []peerasks.URLMetadataAsk,
) map[yacymodel.URLHash]struct{} {
	documents := map[yacymodel.URLHash]struct{}{}
	for _, ask := range asks {
		for _, document := range ask.Documents {
			documents[document] = struct{}{}
		}
	}

	return documents
}

func amountOfAskedDocumentsWithMetadata(
	asks []peerasks.URLMetadataAsk,
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) int {
	askedDocuments := documentsAcrossURLMetadataAsks(asks)
	documentsWithMetadata := map[yacymodel.URLHash]struct{}{}
	for _, answeredAsk := range answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			if _, asked := askedDocuments[metadata.Hash]; !asked {
				continue
			}
			documentsWithMetadata[metadata.Hash] = struct{}{}
		}
	}

	return len(documentsWithMetadata)
}
