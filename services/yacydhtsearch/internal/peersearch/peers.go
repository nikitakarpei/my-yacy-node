// Package peersearch asks many peers one query at once, within a bounded number
// of calls in flight and a per-call time budget. It carries back one answer for
// each peer that replied, and an answer holds no item when the peer had none.
package peersearch

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peersearchwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type Answer struct {
	Peer  yacymodel.Hash
	Items []searchresult.Item
}

type Peers struct {
	wire           peersearchwire.Wire
	inFlight       int
	peerCallBudget time.Duration
}

func New(wire peersearchwire.Wire, inFlight int, peerCallBudget time.Duration) Peers {
	return Peers{wire: wire, inFlight: inFlight, peerCallBudget: peerCallBudget}
}

func (p Peers) Ask(
	ctx context.Context,
	peers []peerdirectory.AskablePeer,
	request yacyproto.SearchRequest,
) []Answer {
	answers := make([]Answer, len(peers))
	replied := make([]bool, len(peers))
	inFlight := make(chan struct{}, p.inFlight)
	var calls sync.WaitGroup

	for index, peer := range peers {
		calls.Add(1)
		go func() {
			defer calls.Done()
			inFlight <- struct{}{}
			defer func() { <-inFlight }()
			answers[index], replied[index] = p.askOne(ctx, peer, request)
		}()
	}
	calls.Wait()

	return answersOfRepliedPeers(answers, replied)
}

func (p Peers) askOne(
	ctx context.Context,
	peer peerdirectory.AskablePeer,
	request yacyproto.SearchRequest,
) (Answer, bool) {
	callCtx, endCall := context.WithTimeout(ctx, p.peerCallBudget)
	defer endCall()

	items, replied := p.wire.Search(callCtx, peer.Address, request)

	return Answer{Peer: peer.Hash, Items: items}, replied
}

func answersOfRepliedPeers(answers []Answer, replied []bool) []Answer {
	keptAnswers := make([]Answer, 0, len(answers))
	for index, answer := range answers {
		if !replied[index] {
			continue
		}
		keptAnswers = append(keptAnswers, answer)
	}

	return keptAnswers
}
