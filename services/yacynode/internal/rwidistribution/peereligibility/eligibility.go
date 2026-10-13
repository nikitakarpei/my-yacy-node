// Package peereligibility tracks which known peers can receive postings now.
// A peer that does not accept an offer is paused. A peer whose
// offers fail too often in a row is excluded for a while, so that the next peer
// takes its place. A peer is busy while its sends in flight, at its usual send
// time, fill the work in flight.
package peereligibility

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Observer interface {
	ObservePeerExcluded()
	ObservePeerSendTime(sendTime time.Duration)
	ObservePeersAtSendLimit(amountOfPeers int)
}

type Config struct {
	LeastPause                  time.Duration
	FailedOffersBeforeExclusion int
	ExclusionDuration           time.Duration
	WorkInFlight                time.Duration
}

type Peers struct {
	mutex      sync.Mutex
	peerStates map[yacymodel.Hash]peerState
	peerLoads  map[yacymodel.Hash]peerLoad
	config     Config
	now        func() time.Time
	observer   Observer
}

type peerState struct {
	pauseEndsAt                time.Time
	exclusionEndsAt            time.Time
	amountOfFailedOffersInARow int
}

func New(config Config, now func() time.Time, observer Observer) *Peers {
	return &Peers{
		peerStates: make(map[yacymodel.Hash]peerState),
		peerLoads:  make(map[yacymodel.Hash]peerLoad),
		config:     config,
		now:        now,
		observer:   observer,
	}
}

func (p *Peers) OfferAccepted(peer yacymodel.Hash) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.peerStates, peer)
}

func (p *Peers) OfferDeclined(peer yacymodel.Hash, requestedPause time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	state := p.peerStates[peer]
	state.pauseEndsAt = p.now().Add(max(requestedPause, p.config.LeastPause))
	p.peerStates[peer] = state
}

func (p *Peers) OfferFailed(ctx context.Context, peer yacymodel.Hash) {
	if !p.recordFailedOffer(peer) {
		return
	}
	slog.WarnContext(
		ctx,
		"peer excluded from distribution after failed offers",
		slog.String("peer", peer.String()),
		slog.Int("failedOffersInARow", p.config.FailedOffersBeforeExclusion),
		slog.Duration("exclusionDuration", p.config.ExclusionDuration),
	)
	p.observer.ObservePeerExcluded()
}

func (p *Peers) recordFailedOffer(peer yacymodel.Hash) (wasExcluded bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := p.now()
	state := p.peerStates[peer]
	state.pauseEndsAt = now.Add(p.config.LeastPause)
	state.amountOfFailedOffersInARow++
	wasExcluded = state.amountOfFailedOffersInARow >= p.config.FailedOffersBeforeExclusion &&
		!now.Before(state.exclusionEndsAt)
	if wasExcluded {
		state.exclusionEndsAt = now.Add(p.config.ExclusionDuration)
	}
	p.peerStates[peer] = state

	return wasExcluded
}

func (p *Peers) IsPaused(peer yacymodel.Hash) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.now().Before(p.peerStates[peer].pauseEndsAt)
}

func (p *Peers) IsExcluded(peer yacymodel.Hash) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.now().Before(p.peerStates[peer].exclusionEndsAt)
}

func (p *Peers) SendStarted(peer yacymodel.Hash) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.peerLoads[peer] = p.peerLoads[peer].withSendStarted()
	p.observer.ObservePeersAtSendLimit(p.amountOfPeersAtSendLimit())
}

func (p *Peers) amountOfPeersAtSendLimit() int {
	amountOfPeers := 0
	for _, load := range p.peerLoads {
		if load.isAtLimitWithin(p.config.WorkInFlight) {
			amountOfPeers++
		}
	}

	return amountOfPeers
}

func (p *Peers) SendEnded(peer yacymodel.Hash, sendTime time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.peerLoads[peer] = p.peerLoads[peer].withSendEndedAfter(sendTime)
	p.observer.ObservePeerSendTime(sendTime)
	p.observer.ObservePeersAtSendLimit(p.amountOfPeersAtSendLimit())
}

func (p *Peers) IsBusy(peer yacymodel.Hash) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.peerLoads[peer].isAtLimitWithin(p.config.WorkInFlight)
}
