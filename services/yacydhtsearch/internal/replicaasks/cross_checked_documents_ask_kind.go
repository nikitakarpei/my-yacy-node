package replicaasks

import (
	"context"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type crossCheckedDocumentsAskKind struct {
	peerCalls  PeerCalls
	hedgeDelay HedgeDelay
}

func (crossCheckedDocumentsAskKind) askedFor() peerasks.AskedFor {
	return peerasks.CrossCheckedDocuments
}

func (crossCheckedDocumentsAskKind) wordPartitionKeyOf(
	ask peerasks.CrossCheckedDocumentsAsk,
) wordPartitionKey {
	return wordPartitionKey{word: ask.Word.String(), partition: ask.Partition}
}

func (kind crossCheckedDocumentsAskKind) hedgeDelayOf(
	ctx context.Context,
	ask peerasks.CrossCheckedDocumentsAsk,
) time.Duration {
	return kind.hedgeDelay.HedgeDelayOf(ctx, ask.Peer)
}

func (kind crossCheckedDocumentsAskKind) putAsk(
	ctx context.Context,
	ask peerasks.CrossCheckedDocumentsAsk,
) (peerasks.AnsweredCrossCheckedDocumentsAsk, bool) {
	answeredAsks := kind.peerCalls.AskForCrossCheckedDocuments(
		ctx,
		[]peerasks.CrossCheckedDocumentsAsk{ask},
	)
	if len(answeredAsks) == 0 {
		return peerasks.AnsweredCrossCheckedDocumentsAsk{}, false
	}

	return answeredAsks[0], true
}

func (kind crossCheckedDocumentsAskKind) isCovering(
	answeredAsk peerasks.AnsweredCrossCheckedDocumentsAsk,
) bool {
	return listsOnlyTheDocumentsTheAskNamed(answeredAsk) &&
		(kind.amountOfDocumentsListedIn(answeredAsk) > 0 || holdsADocumentForTheWord(answeredAsk))
}

func listsOnlyTheDocumentsTheAskNamed(answeredAsk peerasks.AnsweredCrossCheckedDocumentsAsk) bool {
	for _, document := range answeredAsk.DocumentsListedForTheWord {
		if !slices.Contains(answeredAsk.Ask.Documents, document) {
			return false
		}
	}

	return true
}

func holdsADocumentForTheWord(answeredAsk peerasks.AnsweredCrossCheckedDocumentsAsk) bool {
	amountHeld, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get()

	return counted && amountHeld > 0
}

func (crossCheckedDocumentsAskKind) amountOfDocumentsListedIn(
	answeredAsk peerasks.AnsweredCrossCheckedDocumentsAsk,
) int {
	return len(answeredAsk.DocumentsListedForTheWord)
}
