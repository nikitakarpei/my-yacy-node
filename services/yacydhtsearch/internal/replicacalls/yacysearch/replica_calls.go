// Package yacysearch puts the ask of a word partition to one replica through
// the YaCy search endpoint.
package yacysearch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerCalls interface {
	AskForSearchDocuments(
		ctx context.Context,
		asks []peerasks.SearchDocumentsAsk,
	) []peerasks.AnsweredSearchDocumentsAsk
}

type Wants struct {
	Abstract                bool
	MatchedDocumentsCeiling yacymodel.Optional[int]
}

type ReplicaCalls struct {
	peerCalls PeerCalls
	wants     Wants
}

func New(peerCalls PeerCalls, wants Wants) ReplicaCalls {
	return ReplicaCalls{peerCalls: peerCalls, wants: wants}
}

func (calls ReplicaCalls) Put(
	ctx context.Context,
	ask wordpartitionasks.Ask,
	replica peerdirectory.AskablePeer,
) (wordpartitionasks.ReplicaAnswer, bool) {
	answeredAsks := calls.peerCalls.AskForSearchDocuments(
		ctx,
		[]peerasks.SearchDocumentsAsk{calls.searchDocumentsAskFor(ask, replica)},
	)
	if len(answeredAsks) == 0 {
		return wordpartitionasks.ReplicaAnswer{}, false
	}

	return replicaAnswerFrom(answeredAsks[0]), true
}

func (calls ReplicaCalls) searchDocumentsAskFor(
	ask wordpartitionasks.Ask,
	replica peerdirectory.AskablePeer,
) peerasks.SearchDocumentsAsk {
	return peerasks.SearchDocumentsAsk{
		Peer:                    replica,
		Word:                    ask.Word,
		ExcludedWords:           ask.ExcludedWords,
		Language:                ask.Language,
		DocumentsToMatch:        ask.DocumentsToMatch,
		Abstract:                calls.wants.Abstract,
		MatchedDocumentsCeiling: calls.wants.MatchedDocumentsCeiling,
	}
}

func replicaAnswerFrom(
	answeredAsk peerasks.AnsweredSearchDocumentsAsk,
) wordpartitionasks.ReplicaAnswer {
	return wordpartitionasks.ReplicaAnswer{
		Replica:         answeredAsk.Ask.Peer,
		ListedDocuments: listedDocumentsIn(answeredAsk),
		Searched:        answeredAsk.PeerSearched || wantsNoMatchedDocuments(answeredAsk.Ask),
	}
}

func wantsNoMatchedDocuments(ask peerasks.SearchDocumentsAsk) bool {
	return !ask.MatchedDocumentsCeiling.Present()
}
