// Package replicaeligibility tracks which known peers are able to receive
// posting replicas right now, from the outcome of the last offer sent to each.
// A peer that answers anything other than acceptance is held back for a
// cooldown, so the next cycle offers the posting to another peer instead.
package replicaeligibility

import (
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Peers struct {
	mutex         sync.Mutex
	heldBackUntil map[yacymodel.Hash]time.Time
	cooldown      time.Duration
	now           func() time.Time
}

func New(cooldown time.Duration, now func() time.Time) *Peers {
	return &Peers{
		heldBackUntil: make(map[yacymodel.Hash]time.Time),
		cooldown:      cooldown,
		now:           now,
	}
}

func (p *Peers) OfferAccepted(peer yacymodel.Hash) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.heldBackUntil, peer)
}

func (p *Peers) OfferDeclined(peer yacymodel.Hash, requestedPause time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.heldBackUntil[peer] = p.now().Add(max(requestedPause, p.cooldown))
}

func (p *Peers) EligiblePeers(peers []yacymodel.Seed) []yacymodel.Seed {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	eligiblePeers := make([]yacymodel.Seed, 0, len(peers))
	for _, seed := range peers {
		if p.isEligible(seed.Hash) {
			eligiblePeers = append(eligiblePeers, seed)
		}
	}

	return eligiblePeers
}

func (p *Peers) isEligible(peer yacymodel.Hash) bool {
	heldBackUntil, heldBack := p.heldBackUntil[peer]
	if !heldBack {
		return true
	}
	if p.now().Before(heldBackUntil) {
		return false
	}
	delete(p.heldBackUntil, peer)

	return true
}
