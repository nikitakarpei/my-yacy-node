// Package presenceaccrual owns the rule that turns answers into presence. For
// each peer and the address it answered on it holds when that pair first and
// last answered, and the presence the pair has earned by staying reachable
// from one answer to the next. Each answer credits the time since the previous
// one, and no single gap credits more than the continuity limit, so presence
// accrues as vouched-for uptime and never falls. Presence belongs to the pair,
// so a peer hash that answers from another address earns its own presence and
// takes none away. The rule lives here; where the answers are kept and how
// they are replayed belongs to the peer presences that hold them.
package presenceaccrual

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type PresenceAccrualLimits struct {
	Capacity        int
	ContinuityLimit time.Duration
}

type PresenceAccrual struct {
	mutex           sync.Mutex
	observedPeers   *expirable.LRU[PeerAtAddress, ObservedPeer]
	continuityLimit time.Duration
	observer        PresenceAccrualObserver
}

func PresenceAccrualFrom(
	observedPeers []ObservedPeer,
	limits PresenceAccrualLimits,
	observer PresenceAccrualObserver,
) *PresenceAccrual {
	alreadyObserved := expirable.NewLRU[PeerAtAddress, ObservedPeer](limits.Capacity, nil, 0)
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
	peerAnswered PeerAnswered,
) (ObservedPeer, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	answeredPeer, isObserved := p.observedPeers.Get(peerAnswered.PeerAtAddress)
	switch {
	case !isObserved:
		answeredPeer = ObservedPeer{
			PeerAtAddress:    peerAnswered.PeerAtAddress,
			FirstAnsweredAt:  peerAnswered.AnsweredAt,
			LatestAnsweredAt: peerAnswered.AnsweredAt,
		}
	case !peerAnswered.AnsweredAt.After(answeredPeer.LatestAnsweredAt):
		return ObservedPeer{}, false
	default:
		answeredPeer.Presence += min(
			peerAnswered.AnsweredAt.Sub(answeredPeer.LatestAnsweredAt),
			p.continuityLimit,
		)
		answeredPeer.LatestAnsweredAt = peerAnswered.AnsweredAt
	}
	p.observedPeers.Add(peerAnswered.PeerAtAddress, answeredPeer)
	if !isObserved {
		p.observer.PeersObserved(ctx, p.observedPeers.Len())
	}

	return answeredPeer, true
}

func (p *PresenceAccrual) ObservedPeers() []ObservedPeer {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.observedPeers.Values()
}
