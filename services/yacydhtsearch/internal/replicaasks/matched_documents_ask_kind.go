package replicaasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type matchedDocumentsAskKind struct {
	peerCalls  PeerCalls
	hedgeDelay HedgeDelay
}

func (matchedDocumentsAskKind) askedFor() peerasks.AskedFor {
	return peerasks.MatchedDocuments
}

func (matchedDocumentsAskKind) wordPartitionKeyOf(
	ask peerasks.MatchedDocumentsAsk,
) wordPartitionKey {
	return wordPartitionKey{partition: ask.Partition}
}

func (kind matchedDocumentsAskKind) hedgeDelayOf(
	ctx context.Context,
	ask peerasks.MatchedDocumentsAsk,
) time.Duration {
	return kind.hedgeDelay.HedgeDelayOf(ctx, ask.Peer)
}

func (kind matchedDocumentsAskKind) putAsk(
	ctx context.Context,
	ask peerasks.MatchedDocumentsAsk,
) (peerasks.AnsweredMatchedDocumentsAsk, bool) {
	answeredAsks := kind.peerCalls.AskForMatchedDocuments(
		ctx,
		[]peerasks.MatchedDocumentsAsk{ask},
	)
	if len(answeredAsks) == 0 {
		return peerasks.AnsweredMatchedDocumentsAsk{}, false
	}

	return answeredAsks[0], true
}

func (matchedDocumentsAskKind) amountOfDocumentsListedIn(
	answeredAsk peerasks.AnsweredMatchedDocumentsAsk,
) int {
	return len(answeredAsk.MatchedDocuments)
}
