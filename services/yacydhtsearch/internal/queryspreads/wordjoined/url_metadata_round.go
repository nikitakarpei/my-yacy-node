package wordjoined

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func (s Spread) askForURLMetadata(
	ctx context.Context,
	documentsWithoutMetadata map[yacymodel.URLHash]struct{},
	answeredHeldDocumentsAsks []peerasks.AnsweredHeldDocumentsAsk,
) ([]peerasks.URLMetadataAsk, []peerasks.AnsweredURLMetadataAsk) {
	asks := urlMetadataAsksFor(
		documentsWithoutMetadata,
		answeredHeldDocumentsAsks,
		s.metadataDocumentsCeiling,
		s.peersHoldingOneWord,
	)
	secondRound, endSecondRound := contextOfTheSecondRound(ctx)
	defer endSecondRound()

	return asks, s.peerAsks.AskForURLMetadata(secondRound, asks)
}

func contextOfTheSecondRound(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}
