package replicaasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type searchDocumentsAskKind struct {
	peerCalls  PeerCalls
	hedgeDelay HedgeDelay
}

func (searchDocumentsAskKind) wordPartitionKeyOf(
	ask peerasks.SearchDocumentsAsk,
) wordPartitionKey {
	return wordPartitionKey{word: ask.Word.String(), partition: ask.Partition}
}

func (kind searchDocumentsAskKind) hedgeDelayOf(
	ctx context.Context,
	ask peerasks.SearchDocumentsAsk,
) time.Duration {
	return kind.hedgeDelay.HedgeDelayOf(ctx, ask.Peer)
}

func (kind searchDocumentsAskKind) putAsk(
	ctx context.Context,
	ask peerasks.SearchDocumentsAsk,
) (peerasks.AnsweredSearchDocumentsAsk, bool) {
	answeredAsks := kind.peerCalls.AskForSearchDocuments(
		ctx,
		[]peerasks.SearchDocumentsAsk{ask},
	)
	if len(answeredAsks) == 0 {
		return peerasks.AnsweredSearchDocumentsAsk{}, false
	}

	return answeredAsks[0], true
}

func (searchDocumentsAskKind) amountOfDocumentsListedIn(
	answeredAsk peerasks.AnsweredSearchDocumentsAsk,
) int {
	if len(answeredAsk.Abstract) > 0 {
		return len(answeredAsk.Abstract)
	}
	if len(answeredAsk.MatchedDocuments) > 0 {
		return len(answeredAsk.MatchedDocuments)
	}
	amountHeld, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get()
	if !counted {
		return 0
	}

	return max(0, amountHeld)
}
