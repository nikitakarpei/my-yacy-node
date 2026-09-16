// Package presenceaccrual owns the rule that turns answers into presence. For
// each peer and the address it answered on it holds when that pair first and
// last answered, and the presence the pair has earned by staying reachable
// from one answer to the next. Each answer credits the time since the previous
// one, and no single gap credits more than the continuity limit, so presence
// accrues as vouched-for uptime and never falls. Presence belongs to the pair,
// so a peer hash that answers from another address earns its own presence and
// takes none away. The rule lives here; where the answers are kept and how
// they are read again belongs to the peer answer history a presence folds.
package presenceaccrual

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
)

type PresenceAccrualLimits struct {
	Capacity        int
	ContinuityLimit time.Duration
}

type PresenceAccrual struct {
	mutex           sync.Mutex
	observedPeers   *expirable.LRU[peeranswerhistory.PeerAtAddress, ObservedPeer]
	continuityLimit time.Duration
	observer        PresenceAccrualObserver
}

func PresenceAccrualFrom(
	observedPeers []ObservedPeer,
	limits PresenceAccrualLimits,
	observer PresenceAccrualObserver,
) *PresenceAccrual {
	alreadyObserved := expirable.NewLRU[peeranswerhistory.PeerAtAddress, ObservedPeer](
		limits.Capacity, nil, 0,
	)
	for _, observedPeer := range observedPeers {
		alreadyObserved.Add(observedPeer.PeerAtAddress, observedPeer)
	}

	return &PresenceAccrual{
		observedPeers:   alreadyObserved,
		continuityLimit: limits.ContinuityLimit,
		observer:        observer,
	}
}

func (p *PresenceAccrual) Credit(
	ctx context.Context,
	answer peeranswerhistory.PeerAnswer,
) (ObservedPeer, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	answeredPeer, isObserved := p.observedPeers.Get(answer.PeerAtAddress)
	switch {
	case !isObserved:
		answeredPeer = ObservedPeer{
			PeerAtAddress:    answer.PeerAtAddress,
			FirstAnsweredAt:  answer.AnsweredAt,
			LatestAnsweredAt: answer.AnsweredAt,
		}
	case !answer.AnsweredAt.After(answeredPeer.LatestAnsweredAt):
		return ObservedPeer{}, false
	default:
		answeredPeer.Presence += min(
			answer.AnsweredAt.Sub(answeredPeer.LatestAnsweredAt),
			p.continuityLimit,
		)
		answeredPeer.LatestAnsweredAt = answer.AnsweredAt
	}
	p.observedPeers.Add(answer.PeerAtAddress, answeredPeer)
	if !isObserved {
		p.observer.PeersObserved(ctx, p.observedPeers.Len())
	}

	return answeredPeer, true
}

func (p *PresenceAccrual) EarnedPresenceOf(
	peerAtAddress peeranswerhistory.PeerAtAddress,
) time.Duration {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	observedPeer, _ := p.observedPeers.Peek(peerAtAddress)

	return observedPeer.Presence
}

func (p *PresenceAccrual) LatestAnswerOf(
	peerAtAddress peeranswerhistory.PeerAtAddress,
) time.Time {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	observedPeer, _ := p.observedPeers.Peek(peerAtAddress)

	return observedPeer.LatestAnsweredAt
}
