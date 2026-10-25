package peercontact

import (
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

type answeredOrigins struct {
	capacity int

	guard         sync.Mutex
	originPerPeer map[yacymodel.Hash]peerwire.Origin
}

func newAnsweredOrigins(capacity int) *answeredOrigins {
	return &answeredOrigins{
		capacity:      capacity,
		originPerPeer: make(map[yacymodel.Hash]peerwire.Origin),
	}
}

func (a *answeredOrigins) record(peer yacymodel.Hash, origin peerwire.Origin) {
	a.guard.Lock()
	defer a.guard.Unlock()

	if _, known := a.originPerPeer[peer]; !known && len(a.originPerPeer) >= a.capacity {
		for evictedPeer := range a.originPerPeer {
			delete(a.originPerPeer, evictedPeer)

			break
		}
	}
	a.originPerPeer[peer] = origin
}

func (a *answeredOrigins) forget(peer yacymodel.Hash) {
	a.guard.Lock()
	defer a.guard.Unlock()

	delete(a.originPerPeer, peer)
}

func (a *answeredOrigins) originOf(peer yacymodel.Hash) (peerwire.Origin, bool) {
	a.guard.Lock()
	defer a.guard.Unlock()

	origin, found := a.originPerPeer[peer]

	return origin, found
}
