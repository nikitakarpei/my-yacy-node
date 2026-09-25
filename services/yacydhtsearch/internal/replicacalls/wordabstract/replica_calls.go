// Package wordabstract puts one ask for the abstract of a word to the replica
// it addresses, and tells what the answer says about the word partition.
package wordabstract

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
)

type PeerCalls interface {
	AskForWordAbstracts(
		ctx context.Context,
		asks []peerasks.WordAbstractAsk,
	) []peerasks.AnsweredWordAbstractAsk
}

type ReplicaCalls struct {
	peerCalls PeerCalls
}

func New(peerCalls PeerCalls) ReplicaCalls {
	return ReplicaCalls{peerCalls: peerCalls}
}

func (ReplicaCalls) ReplicaOf(ask peerasks.WordAbstractAsk) replicaasks.Replica {
	return replicaasks.Replica{Peer: ask.Peer, Word: ask.Word, Partition: ask.Partition}
}

func (replicaCalls ReplicaCalls) AnswerTo(
	ctx context.Context,
	ask peerasks.WordAbstractAsk,
) (peerasks.AnsweredWordAbstractAsk, bool) {
	answeredAsks := replicaCalls.peerCalls.AskForWordAbstracts(
		ctx,
		[]peerasks.WordAbstractAsk{ask},
	)
	if len(answeredAsks) == 0 {
		return peerasks.AnsweredWordAbstractAsk{}, false
	}

	return answeredAsks[0], true
}

func (ReplicaCalls) CoverageFrom(answer peerasks.AnsweredWordAbstractAsk) replicaasks.Coverage {
	return replicaasks.Coverage{AmountOfDocumentsListed: len(answer.Abstract), PeerSearched: true}
}
