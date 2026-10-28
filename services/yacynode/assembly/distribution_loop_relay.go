package yacynodeassembly

import (
	"sync/atomic"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
)

type distributionSignals interface {
	ObservePostingAdmitted(position yacymodel.DHTRingPosition)
	ObserveReachablePeers(amountOfPeers int)
}

type distributionLoopRelay struct {
	loop atomic.Pointer[distributionSignals]
}

func (r *distributionLoopRelay) forwardTo(loop distributionSignals) {
	r.loop.Store(&loop)
}

func (r *distributionLoopRelay) ObservePostingAdmitted(position yacymodel.DHTRingPosition) {
	if loop := r.loop.Load(); loop != nil {
		(*loop).ObservePostingAdmitted(position)
	}
}

func (r *distributionLoopRelay) ObserveReachablePeers(amountOfPeers int) {
	if loop := r.loop.Load(); loop != nil {
		(*loop).ObserveReachablePeers(amountOfPeers)
	}
}

type rosterObserverFanOut struct {
	metrics peerroster.RosterObserver
	relay   *distributionLoopRelay
}

func (f rosterObserverFanOut) ObserveKnownPeers(count int) {
	f.metrics.ObserveKnownPeers(count)
}

func (f rosterObserverFanOut) ObserveReachablePeers(count int) {
	f.metrics.ObserveReachablePeers(count)
	f.relay.ObserveReachablePeers(count)
}

func (f rosterObserverFanOut) ObserveDroppedSeed(reason peerroster.SeedDropReason) {
	f.metrics.ObserveDroppedSeed(reason)
}
