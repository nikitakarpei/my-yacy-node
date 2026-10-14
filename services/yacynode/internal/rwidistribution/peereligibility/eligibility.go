// Package peereligibility tracks which known peers are able to receive
// postings right now, from the outcome of the offers sent to each.
// A peer that does not accept an offer goes into a cooldown, and postings that
// need it wait until the cooldown ends. A peer whose offers fail too often in a
// row is excluded for a while, so that the next peer takes its place.
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
}

type Config struct {
	Cooldown                    time.Duration
	FailedOffersBeforeExclusion int
	ExclusionDuration           time.Duration
}

type Peers struct {
	mutex      sync.Mutex
	peerStates map[yacymodel.Hash]peerState
	config     Config
	now        func() time.Time
	observer   Observer
}

type peerState struct {
	cooldownEndsAt             time.Time
	exclusionEndsAt            time.Time
	amountOfFailedOffersInARow int
}

func New(config Config, now func() time.Time, observer Observer) *Peers {
	return &Peers{
		peerStates: make(map[yacymodel.Hash]peerState),
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
	state.cooldownEndsAt = p.now().Add(max(requestedPause, p.config.Cooldown))
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
	state.cooldownEndsAt = now.Add(p.config.Cooldown)
	state.amountOfFailedOffersInARow++
	wasExcluded = state.amountOfFailedOffersInARow >= p.config.FailedOffersBeforeExclusion &&
		!now.Before(state.exclusionEndsAt)
	if wasExcluded {
		state.exclusionEndsAt = now.Add(p.config.ExclusionDuration)
	}
	p.peerStates[peer] = state

	return wasExcluded
}

func (p *Peers) IsInCooldown(peer yacymodel.Hash) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.now().Before(p.peerStates[peer].cooldownEndsAt)
}

func (p *Peers) IsExcluded(peer yacymodel.Hash) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.now().Before(p.peerStates[peer].exclusionEndsAt)
}
