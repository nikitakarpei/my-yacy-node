package replicaasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type matchedAndHeldDocumentsAskKind struct {
	peerCalls  PeerCalls
	hedgeDelay HedgeDelay
}

func (matchedAndHeldDocumentsAskKind) wordPartitionKeyOf(
	ask peerasks.MatchedAndHeldDocumentsAsk,
) wordPartitionKey {
	return wordPartitionKey{word: ask.Word.String(), partition: ask.Partition}
}

func (kind matchedAndHeldDocumentsAskKind) hedgeDelayOf(
	ctx context.Context,
	ask peerasks.MatchedAndHeldDocumentsAsk,
) time.Duration {
	return kind.hedgeDelay.HedgeDelayOf(ctx, ask.Peer)
}

func (kind matchedAndHeldDocumentsAskKind) putAsk(
	ctx context.Context,
	ask peerasks.MatchedAndHeldDocumentsAsk,
) (peerasks.AnsweredMatchedAndHeldDocumentsAsk, bool) {
	answeredAsks := kind.peerCalls.AskForMatchedAndHeldDocuments(
		ctx,
		[]peerasks.MatchedAndHeldDocumentsAsk{ask},
	)
	if len(answeredAsks) == 0 {
		return peerasks.AnsweredMatchedAndHeldDocumentsAsk{}, false
	}

	return answeredAsks[0], true
}

func (matchedAndHeldDocumentsAskKind) amountOfDocumentsListedIn(
	answeredAsk peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) int {
	if len(answeredAsk.Abstract) > 0 {
		return len(answeredAsk.Abstract)
	}
	amountHeld, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get()
	if !counted {
		return 0
	}

	return max(0, amountHeld)
}
