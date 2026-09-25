// Package searchdocuments puts one ask for search documents to the replica it
// addresses, and tells what the answer says about the word partition.
package searchdocuments

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
)

type PeerCalls interface {
	AskForSearchDocuments(
		ctx context.Context,
		asks []peerasks.SearchDocumentsAsk,
	) []peerasks.AnsweredSearchDocumentsAsk
}

type ReplicaCalls struct {
	peerCalls PeerCalls
}

func New(peerCalls PeerCalls) ReplicaCalls {
	return ReplicaCalls{peerCalls: peerCalls}
}

func (ReplicaCalls) ReplicaOf(ask peerasks.SearchDocumentsAsk) replicaasks.Replica {
	return replicaasks.Replica{Peer: ask.Peer, Word: ask.Word, Partition: ask.Partition}
}

func (replicaCalls ReplicaCalls) AnswerTo(
	ctx context.Context,
	ask peerasks.SearchDocumentsAsk,
) (peerasks.AnsweredSearchDocumentsAsk, bool) {
	answeredAsks := replicaCalls.peerCalls.AskForSearchDocuments(
		ctx,
		[]peerasks.SearchDocumentsAsk{ask},
	)
	if len(answeredAsks) == 0 {
		return peerasks.AnsweredSearchDocumentsAsk{}, false
	}

	return answeredAsks[0], true
}

func (ReplicaCalls) CoverageFrom(answer peerasks.AnsweredSearchDocumentsAsk) replicaasks.Coverage {
	return replicaasks.Coverage{
		AmountOfDocumentsListed: amountOfDocumentsListedIn(answer),
		PeerSearched:            answer.PeerSearched,
	}
}

func amountOfDocumentsListedIn(answer peerasks.AnsweredSearchDocumentsAsk) int {
	if len(answer.Abstract) > 0 {
		return len(answer.Abstract)
	}
	if len(answer.MatchedDocuments) > 0 {
		return len(answer.MatchedDocuments)
	}
	amountHeld, counted := answer.AmountOfDocumentsHeldForTheWord.Get()
	if !counted {
		return 0
	}

	return max(0, amountHeld)
}
