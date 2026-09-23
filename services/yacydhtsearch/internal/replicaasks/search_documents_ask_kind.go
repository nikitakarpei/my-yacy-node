package replicaasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type searchDocumentsAskKind struct {
	peerCalls  PeerCalls
	hedgeDelay HedgeDelay
}

func (searchDocumentsAskKind) askedFor() peerasks.AskedFor {
	return peerasks.SearchDocuments
}

func (searchDocumentsAskKind) partitionKeyOf(
	ask peerasks.SearchDocumentsAsk,
) partitionKey {
	return partitionKey{word: ask.Word.String(), partition: ask.Partition}
}

func (searchDocumentsAskKind) peerOf(ask peerasks.SearchDocumentsAsk) yacymodel.Hash {
	return ask.Peer.Hash
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

func (kind searchDocumentsAskKind) isCovering(
	answeredAsk peerasks.AnsweredSearchDocumentsAsk,
) bool {
	return !answeredAsk.IgnoredTheDocumentsToMatch() &&
		kind.amountOfDocumentsListedIn(answeredAsk) > 0
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
