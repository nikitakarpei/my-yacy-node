// Package replicarecipients tracks which known peers are able to receive
// posting replicas right now, from the outcome of the last offer sent to each.
// A peer that answers anything other than acceptance is held back for a
// cooldown, so the next cycle offers the posting to another peer instead.
package replicarecipients

import (
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
)

type Recipients struct {
	mutex    sync.Mutex
	heldBack map[yacymodel.Hash]time.Time
	cooldown time.Duration
	now      func() time.Time
}

func New(cooldown time.Duration, now func() time.Time) *Recipients {
	return &Recipients{
		heldBack: make(map[yacymodel.Hash]time.Time),
		cooldown: cooldown,
		now:      now,
	}
}

// OfferAnswered records how a peer answered a posting offer. An answer other
// than acceptance holds the peer back until the longer of the peer's own
// requested pause and the configured cooldown has passed.
func (r *Recipients) OfferAnswered(
	peer yacymodel.Hash,
	outcome postingcourier.Outcome,
	requestedPause time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if outcome == postingcourier.Accepted {
		delete(r.heldBack, peer)

		return
	}
	r.heldBack[peer] = r.now().Add(max(requestedPause, r.cooldown))
}

// Eligible reports whether a peer can receive a replica now, and forgets a peer
// whose cooldown has passed.
func (r *Recipients) Eligible(peer yacymodel.Hash) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	until, held := r.heldBack[peer]
	if !held {
		return true
	}
	if r.now().Before(until) {
		return false
	}
	delete(r.heldBack, peer)

	return true
}
