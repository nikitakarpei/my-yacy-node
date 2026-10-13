// Package replicaeligibility tracks which known peers are able to receive
// posting replicas right now, from the outcome of the last offer sent to each.
// A peer that answers anything other than acceptance goes into a cooldown, and
// postings that need it wait until the cooldown ends.
package replicaeligibility

import (
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Peers struct {
	mutex          sync.Mutex
	cooldownEndsAt map[yacymodel.Hash]time.Time
	cooldown       time.Duration
	now            func() time.Time
}

func New(cooldown time.Duration, now func() time.Time) *Peers {
	return &Peers{
		cooldownEndsAt: make(map[yacymodel.Hash]time.Time),
		cooldown:       cooldown,
		now:            now,
	}
}

func (p *Peers) OfferAccepted(peer yacymodel.Hash) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.cooldownEndsAt, peer)
}

func (p *Peers) OfferDeclined(peer yacymodel.Hash, requestedPause time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.cooldownEndsAt[peer] = p.now().Add(max(requestedPause, p.cooldown))
}

func (p *Peers) CooldownLeftOf(peer yacymodel.Hash) time.Duration {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	cooldownEndsAt, inCooldown := p.cooldownEndsAt[peer]
	if !inCooldown {
		return 0
	}
	cooldownLeft := cooldownEndsAt.Sub(p.now())
	if cooldownLeft <= 0 {
		delete(p.cooldownEndsAt, peer)

		return 0
	}

	return cooldownLeft
}
