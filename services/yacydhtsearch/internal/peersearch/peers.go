// Package peersearch asks many peers one query at once, within a bounded number
// of calls in flight and a per-call time budget.
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
	inFlight := make(chan struct{}, p.inFlight)
	var calls sync.WaitGroup

	for index, peer := range peers {
		calls.Add(1)
		go func() {
			defer calls.Done()
			inFlight <- struct{}{}
			defer func() { <-inFlight }()
			answers[index] = p.askOne(ctx, peer, request)
		}()
	}
	calls.Wait()

	return answersWithItems(answers)
}

func (p Peers) askOne(
	ctx context.Context,
	peer peerdirectory.AskablePeer,
	request yacyproto.SearchRequest,
) Answer {
	callCtx, endCall := context.WithTimeout(ctx, p.peerCallBudget)
	defer endCall()

	return Answer{
		Peer:  peer.Hash,
		Items: p.wire.Search(callCtx, peer.Address, request),
	}
}

func answersWithItems(answers []Answer) []Answer {
	keptAnswers := make([]Answer, 0, len(answers))
	for _, answer := range answers {
		if len(answer.Items) == 0 {
			continue
		}
		keptAnswers = append(keptAnswers, answer)
	}

	return keptAnswers
}
