// Package peerpresence owns what this deployment has observed of every peer it
// probes: for each peer and the address it answered on, when that pair first
// and last answered, and the presence the pair has earned by staying reachable
// from one answer to the next. Presence belongs to the pair, so a peer hash
// that answers from another address earns its own presence and takes none
// away. The rule that grows presence lives here; where the answers are kept
// and how they are replayed belongs to the histories that hold them.
package peerpresence

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type PeerPresenceLimits struct {
	Capacity        int
	ContinuityLimit time.Duration
}

type PeerPresence struct {
	mutex           sync.Mutex
	observedPeers   *expirable.LRU[PeerAtAddress, ObservedPeer]
	continuityLimit time.Duration
	observer        PeerPresenceObserver
}

func PeerPresenceFrom(
	observedPeers []ObservedPeer,
	limits PeerPresenceLimits,
	observer PeerPresenceObserver,
) *PeerPresence {
	alreadyObserved := expirable.NewLRU[PeerAtAddress, ObservedPeer](limits.Capacity, nil, 0)
	for _, observedPeer := range observedPeers {
		alreadyObserved.Add(observedPeer.PeerAtAddress, observedPeer)
	}

	return &PeerPresence{
		observedPeers:   alreadyObserved,
		continuityLimit: limits.ContinuityLimit,
		observer:        observer,
	}
}

func (p *PeerPresence) Credit(ctx context.Context, peerAnswered PeerAnswered) (ObservedPeer, bool) {
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

func (p *PeerPresence) ObservedPeers() []ObservedPeer {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.observedPeers.Values()
}
