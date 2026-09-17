// Package replicaasks puts the asks of a word partition to its replicas in
// turn, and settles the partition as soon as enough replicas have listed
// documents for the word.
package replicaasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type PeerCalls interface {
	AskForMatchedAndHeldDocuments(
		ctx context.Context,
		asks []peerasks.MatchedAndHeldDocumentsAsk,
	) []peerasks.AnsweredMatchedAndHeldDocumentsAsk
	AskForMatchedDocuments(
		ctx context.Context,
		asks []peerasks.MatchedDocumentsAsk,
	) []peerasks.AnsweredMatchedDocumentsAsk
}

type HedgeDelay interface {
	HedgeDelayOf(ctx context.Context, peer peerdirectory.AskablePeer) time.Duration
}

type Asks struct {
	peerCalls                  PeerCalls
	hedgeDelay                 HedgeDelay
	replicasCoveringAPartition int
	observer                   ReplicaAsksObserver
}

func New(
	peerCalls PeerCalls,
	hedgeDelay HedgeDelay,
	replicasCoveringAPartition int,
	observer ReplicaAsksObserver,
) Asks {
	return Asks{
		peerCalls:                  peerCalls,
		hedgeDelay:                 hedgeDelay,
		replicasCoveringAPartition: replicasCoveringAPartition,
		observer:                   observer,
	}
}

type matchedAndHeldDocumentsReplicaCalls = replicaCalls[
	peerasks.MatchedAndHeldDocumentsAsk,
	peerasks.AnsweredMatchedAndHeldDocumentsAsk,
]

func (asks Asks) AskForMatchedAndHeldDocuments(
	ctx context.Context,
	asksInReplicaOrder []peerasks.MatchedAndHeldDocumentsAsk,
) []peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	return answersOfWordPartitions(
		ctx,
		asksInReplicaOrder,
		matchedAndHeldDocumentsReplicaCalls{
			wordPartitionKeyOf:        wordPartitionKeyOfTheMatchedAndHeldDocumentsAsk,
			hedgeDelayOf:              asks.hedgeDelayOfTheMatchedAndHeldDocumentsAsk,
			putAsk:                    asks.putTheMatchedAndHeldDocumentsAsk,
			amountOfDocumentsListedIn: amountOfDocumentsListedForTheWordIn,
		},
		asks.replicasCoveringAPartition,
		asks.observer,
	)
}

func wordPartitionKeyOfTheMatchedAndHeldDocumentsAsk(
	ask peerasks.MatchedAndHeldDocumentsAsk,
) wordPartitionKey {
	return wordPartitionKey{word: ask.Word.String(), partition: ask.Partition}
}

func (asks Asks) hedgeDelayOfTheMatchedAndHeldDocumentsAsk(
	ctx context.Context,
	ask peerasks.MatchedAndHeldDocumentsAsk,
) time.Duration {
	return asks.hedgeDelay.HedgeDelayOf(ctx, ask.Peer)
}

func (asks Asks) putTheMatchedAndHeldDocumentsAsk(
	ctx context.Context,
	ask peerasks.MatchedAndHeldDocumentsAsk,
) (peerasks.AnsweredMatchedAndHeldDocumentsAsk, bool) {
	answeredAsks := asks.peerCalls.AskForMatchedAndHeldDocuments(
		ctx,
		[]peerasks.MatchedAndHeldDocumentsAsk{ask},
	)
	if len(answeredAsks) == 0 {
		return peerasks.AnsweredMatchedAndHeldDocumentsAsk{}, false
	}

	return answeredAsks[0], true
}

func amountOfDocumentsListedForTheWordIn(
	answeredAsk peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) int {
	if len(answeredAsk.DocumentsListedForTheWord) > 0 {
		return len(answeredAsk.DocumentsListedForTheWord)
	}
	amountHeld, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get()
	if !counted {
		return 0
	}

	return max(0, amountHeld)
}

type matchedDocumentsReplicaCalls = replicaCalls[
	peerasks.MatchedDocumentsAsk,
	peerasks.AnsweredMatchedDocumentsAsk,
]

func (asks Asks) AskForMatchedDocuments(
	ctx context.Context,
	asksInReplicaOrder []peerasks.MatchedDocumentsAsk,
) []peerasks.AnsweredMatchedDocumentsAsk {
	return answersOfWordPartitions(
		ctx,
		asksInReplicaOrder,
		matchedDocumentsReplicaCalls{
			wordPartitionKeyOf:        wordPartitionKeyOfTheMatchedDocumentsAsk,
			hedgeDelayOf:              asks.hedgeDelayOfTheMatchedDocumentsAsk,
			putAsk:                    asks.putTheMatchedDocumentsAsk,
			amountOfDocumentsListedIn: amountOfMatchedDocumentsIn,
		},
		asks.replicasCoveringAPartition,
		asks.observer,
	)
}

func wordPartitionKeyOfTheMatchedDocumentsAsk(ask peerasks.MatchedDocumentsAsk) wordPartitionKey {
	return wordPartitionKey{partition: ask.Partition}
}

func (asks Asks) hedgeDelayOfTheMatchedDocumentsAsk(
	ctx context.Context,
	ask peerasks.MatchedDocumentsAsk,
) time.Duration {
	return asks.hedgeDelay.HedgeDelayOf(ctx, ask.Peer)
}

func (asks Asks) putTheMatchedDocumentsAsk(
	ctx context.Context,
	ask peerasks.MatchedDocumentsAsk,
) (peerasks.AnsweredMatchedDocumentsAsk, bool) {
	answeredAsks := asks.peerCalls.AskForMatchedDocuments(
		ctx,
		[]peerasks.MatchedDocumentsAsk{ask},
	)
	if len(answeredAsks) == 0 {
		return peerasks.AnsweredMatchedDocumentsAsk{}, false
	}

	return answeredAsks[0], true
}

func amountOfMatchedDocumentsIn(answeredAsk peerasks.AnsweredMatchedDocumentsAsk) int {
	return len(answeredAsk.MatchedDocuments)
}
