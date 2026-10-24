// Package peereligibility tracks which known peers can receive postings now.
// A peer that does not accept an offer is paused. A peer that does not accept
// too many offers in a row is excluded for a while, so that the next peer
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
	ObservePeerExcluded(cause string)
	ObservePeerSendTime(sendTime time.Duration)
	ObservePeersAtSendLimit(amountOfPeers int)
}

type Config struct {
	LeastPause                      time.Duration
	UnacceptedOffersBeforeExclusion int
	ExclusionDuration               time.Duration
	WorkInFlight                    time.Duration
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
	pauseEndsAt                    time.Time
	pauseCause                     PauseCause
	exclusionEndsAt                time.Time
	amountOfUnacceptedOffersInARow int
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

func (p *Peers) OfferDeclined(
	ctx context.Context,
	peer yacymodel.Hash,
	requestedPause time.Duration,
) {
	pause := max(requestedPause, p.config.LeastPause)
	if p.recordUnacceptedOffer(peer, pause, PauseDeclined) {
		p.reportExclusion(ctx, peer, PauseDeclined)
	}
}

func (p *Peers) OfferFailed(ctx context.Context, peer yacymodel.Hash) {
	if p.recordUnacceptedOffer(peer, p.config.LeastPause, PauseFailed) {
		p.reportExclusion(ctx, peer, PauseFailed)
	}
}

func (p *Peers) recordUnacceptedOffer(
	peer yacymodel.Hash,
	pause time.Duration,
	cause PauseCause,
) (wasExcluded bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := p.now()
	state := p.peerStates[peer]
	state.pauseEndsAt = now.Add(pause)
	state.pauseCause = cause
	state.amountOfUnacceptedOffersInARow++
	wasExcluded = state.amountOfUnacceptedOffersInARow >= p.config.UnacceptedOffersBeforeExclusion &&
		!now.Before(state.exclusionEndsAt)
	if wasExcluded {
		state.exclusionEndsAt = now.Add(p.config.ExclusionDuration)
	}
	p.peerStates[peer] = state

	return wasExcluded
}

func (p *Peers) reportExclusion(ctx context.Context, peer yacymodel.Hash, cause PauseCause) {
	slog.WarnContext(
		ctx,
		"peer excluded from distribution after unaccepted offers",
		slog.String("peer", peer.String()),
		slog.String("cause", string(cause)),
		slog.Int("unacceptedOffersInARow", p.config.UnacceptedOffersBeforeExclusion),
		slog.Duration("exclusionDuration", p.config.ExclusionDuration),
	)
	p.observer.ObservePeerExcluded(string(cause))
}

func (p *Peers) IsPaused(peer yacymodel.Hash) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.now().Before(p.peerStates[peer].pauseEndsAt)
}

func (p *Peers) PauseOf(peer yacymodel.Hash) (Pause, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	state := p.peerStates[peer]
	if !p.now().Before(state.pauseEndsAt) {
		return Pause{}, false
	}

	return Pause{EndsAt: state.pauseEndsAt, Cause: state.pauseCause}, true
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
