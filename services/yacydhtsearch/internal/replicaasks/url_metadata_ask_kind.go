package replicaasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type urlMetadataAskKind struct {
	peerCalls  PeerCalls
	hedgeDelay HedgeDelay
}

func (urlMetadataAskKind) askedFor() peerasks.AskedFor {
	return peerasks.URLMetadata
}

func (urlMetadataAskKind) partitionKeyOf(ask peerasks.URLMetadataAsk) partitionKey {
	return partitionKey{partition: ask.Partition}
}

func (urlMetadataAskKind) peerOf(ask peerasks.URLMetadataAsk) yacymodel.Hash {
	return ask.Peer.Hash
}

func (kind urlMetadataAskKind) hedgeDelayOf(
	ctx context.Context,
	ask peerasks.URLMetadataAsk,
) time.Duration {
	return kind.hedgeDelay.HedgeDelayOf(ctx, ask.Peer)
}

func (kind urlMetadataAskKind) putAsk(
	ctx context.Context,
	ask peerasks.URLMetadataAsk,
) (peerasks.AnsweredURLMetadataAsk, bool) {
	answeredAsks := kind.peerCalls.AskForURLMetadata(ctx, []peerasks.URLMetadataAsk{ask})
	if len(answeredAsks) == 0 {
		return peerasks.AnsweredURLMetadataAsk{}, false
	}

	return answeredAsks[0], true
}

func (urlMetadataAskKind) isCovering(answeredAsk peerasks.AnsweredURLMetadataAsk) bool {
	documentsWithMetadata := make(
		map[yacymodel.URLHash]struct{},
		len(answeredAsk.MetadataOfEachDocument),
	)
	for _, metadata := range answeredAsk.MetadataOfEachDocument {
		documentsWithMetadata[metadata.Hash] = struct{}{}
	}
	for _, document := range answeredAsk.Ask.Documents {
		if _, withMetadata := documentsWithMetadata[document]; !withMetadata {
			return false
		}
	}

	return true
}

func (urlMetadataAskKind) amountOfDocumentsListedIn(
	answeredAsk peerasks.AnsweredURLMetadataAsk,
) int {
	return len(answeredAsk.MetadataOfEachDocument)
}
