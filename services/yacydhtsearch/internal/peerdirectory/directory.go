package peerdirectory

import (
	"context"
	"maps"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type StalePeerSource interface {
	StalestPeersFirst(
		ctx context.Context,
		members []KnownPeer,
		candidates []CandidatePeer,
	) []yacymodel.Hash
}

type DirectoryObserver interface {
	PeerAdmitted(ctx context.Context, peer yacymodel.Hash, addresses int)
	PeerAnswered(
		ctx context.Context,
		peer yacymodel.Hash,
		address string,
		answeredAt time.Time,
	)
	PeerWentSilent(ctx context.Context, peer yacymodel.Hash)
	PeerDropped(ctx context.Context, peer yacymodel.Hash)
	PeersKnown(ctx context.Context, amountOfKnownPeers, amountOfAnsweringPeers, capacity int)
}

type DirectoryLimits struct {
	Capacity      int
	Cooldown      time.Duration
	NewcomerShare float64
}

type Directory struct {
	mutex    sync.Mutex
	peers    map[yacymodel.Hash]KnownPeer
	limits   DirectoryLimits
	now      func() time.Time
	stale    StalePeerSource
	observer DirectoryObserver
}

func New(
	limits DirectoryLimits,
	now func() time.Time,
	stale StalePeerSource,
	observer DirectoryObserver,
) *Directory {
	return &Directory{
		peers:    make(map[yacymodel.Hash]KnownPeer, limits.Capacity),
		limits:   limits,
		now:      now,
		stale:    stale,
		observer: observer,
	}
}

func (d *Directory) Admit(ctx context.Context, seeds []yacymodel.Seed) {
	admittedPeers, droppedPeers := d.holdAdmittedPeers(ctx, seeds)
	for _, droppedPeer := range droppedPeers {
		d.observer.PeerDropped(ctx, droppedPeer)
	}
	for _, admittedPeer := range admittedPeers {
		d.observer.PeerAdmitted(ctx, admittedPeer.Hash, len(admittedPeer.Addresses))
	}
	d.reportPeersKnown(ctx)
}

func (d *Directory) holdAdmittedPeers(
	ctx context.Context,
	seeds []yacymodel.Seed,
) ([]KnownPeer, []yacymodel.Hash) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.refreshTheAddressesOfHeldPeers(seeds)
	candidates := d.candidatesAmong(seeds)
	refusedCandidates, droppedPeers := d.peersThatDoNotFit(ctx, candidates)
	d.drop(droppedPeers)

	return d.hold(candidates, refusedCandidates), droppedPeers
}

func (d *Directory) refreshTheAddressesOfHeldPeers(seeds []yacymodel.Seed) {
	for _, seed := range seeds {
		addresses := addressesOf(seed)
		held, isHeld := d.peers[seed.Hash]
		if len(addresses) == 0 || !isHeld {
			continue
		}
		held.Addresses = addressesLedBy(held.AnsweredAddress, addresses)
		d.peers[seed.Hash] = held
	}
}

func (d *Directory) candidatesAmong(seeds []yacymodel.Seed) []CandidatePeer {
	var candidates []CandidatePeer
	offered := map[yacymodel.Hash]struct{}{}
	for _, seed := range seeds {
		addresses := addressesOf(seed)
		_, isHeld := d.peers[seed.Hash]
		_, isOffered := offered[seed.Hash]
		if len(addresses) == 0 || isHeld || isOffered {
			continue
		}
		offered[seed.Hash] = struct{}{}
		candidates = append(candidates, CandidatePeer{Hash: seed.Hash, Addresses: addresses})
	}

	return candidates
}

func (d *Directory) peersThatDoNotFit(
	ctx context.Context,
	candidates []CandidatePeer,
) (map[yacymodel.Hash]struct{}, []yacymodel.Hash) {
	peersOverTheCapacity := len(d.peers) + len(candidates) - d.limits.Capacity
	if peersOverTheCapacity <= 0 {
		return nil, nil
	}
	newcomers := d.newcomersDrawnAmong(candidates)
	offered := hashesOf(candidates)
	refusedCandidates := map[yacymodel.Hash]struct{}{}
	var droppedPeers []yacymodel.Hash
	for _, peer := range d.stalestPeersFirst(ctx, candidates) {
		if peersOverTheCapacity == 0 {
			break
		}
		if _, drawn := newcomers[peer]; drawn {
			continue
		}
		if _, isCandidate := offered[peer]; isCandidate {
			refusedCandidates[peer] = struct{}{}
		} else {
			droppedPeers = append(droppedPeers, peer)
		}
		peersOverTheCapacity--
	}

	return refusedCandidates, droppedPeers
}

func (d *Directory) newcomersDrawnAmong(
	candidates []CandidatePeer,
) map[yacymodel.Hash]struct{} {
	amountDrawn := min(
		len(candidates),
		int(float64(d.limits.Capacity)*d.limits.NewcomerShare),
	)
	drawn := make(map[yacymodel.Hash]struct{}, amountDrawn)
	//nolint:gosec // G404: which new peers a directory admits needs no unpredictability.
	for _, index := range rand.Perm(len(candidates))[:amountDrawn] {
		drawn[candidates[index].Hash] = struct{}{}
	}

	return drawn
}

func hashesOf(candidates []CandidatePeer) map[yacymodel.Hash]struct{} {
	hashes := make(map[yacymodel.Hash]struct{}, len(candidates))
	for _, candidate := range candidates {
		hashes[candidate.Hash] = struct{}{}
	}

	return hashes
}

func (d *Directory) stalestPeersFirst(
	ctx context.Context,
	candidates []CandidatePeer,
) []yacymodel.Hash {
	return d.stale.StalestPeersFirst(ctx, slices.Collect(maps.Values(d.peers)), candidates)
}

func (d *Directory) drop(peers []yacymodel.Hash) {
	for _, peer := range peers {
		delete(d.peers, peer)
	}
}

func (d *Directory) hold(
	candidates []CandidatePeer,
	refusedCandidates map[yacymodel.Hash]struct{},
) []KnownPeer {
	admittedPeers := make([]KnownPeer, 0, len(candidates)-len(refusedCandidates))
	for _, candidate := range candidates {
		if _, refused := refusedCandidates[candidate.Hash]; refused {
			continue
		}
		d.peers[candidate.Hash] = KnownPeer{
			Hash:       candidate.Hash,
			Addresses:  candidate.Addresses,
			AdmittedAt: d.now(),
		}
		admittedPeers = append(admittedPeers, d.peers[candidate.Hash])
	}

	return admittedPeers
}

func (d *Directory) KnownPeers(ctx context.Context) []KnownPeer {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return slices.Collect(maps.Values(d.peers))
}

func (d *Directory) AskablePeers(ctx context.Context) []AskablePeer {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	askable := make([]AskablePeer, 0, len(d.peers))
	for _, peer := range d.peers {
		if !peer.answersNow() || d.now().Sub(peer.ChosenAt) < d.limits.Cooldown {
			continue
		}
		askable = append(askable, AskablePeer{Hash: peer.Hash, Address: peer.AnsweredAddress})
	}

	return askable
}

func (d *Directory) MarkPeersChosen(ctx context.Context, peers []AskablePeer) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for _, chosenPeer := range peers {
		known, ok := d.peers[chosenPeer.Hash]
		if !ok {
			continue
		}
		known.ChosenAt = d.now()
		d.peers[chosenPeer.Hash] = known
	}
}

func (d *Directory) ConfirmAnswering(ctx context.Context, peer yacymodel.Hash, address string) {
	answeredAt, wasSilent, isKnown := d.holdAnswer(peer, address)
	if !isKnown {
		return
	}
	d.observer.PeerAnswered(ctx, peer, address, answeredAt)
	if wasSilent {
		d.reportPeersKnown(ctx)
	}
}

func (d *Directory) holdAnswer(
	peer yacymodel.Hash,
	address string,
) (time.Time, bool, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	known, ok := d.peers[peer]
	if !ok {
		return time.Time{}, false, false
	}
	wasSilent := !known.answersNow()
	known.AnsweredAddress = address
	known.AnsweredAt = d.now()
	d.peers[peer] = known

	return known.AnsweredAt, wasSilent, true
}

func (d *Directory) ConfirmSilent(ctx context.Context, peer yacymodel.Hash) {
	wasAnswering, isKnown := d.holdSilence(peer)
	if !isKnown {
		return
	}
	d.observer.PeerWentSilent(ctx, peer)
	if wasAnswering {
		d.reportPeersKnown(ctx)
	}
}

func (d *Directory) holdSilence(peer yacymodel.Hash) (bool, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	known, ok := d.peers[peer]
	if !ok {
		return false, false
	}
	wasAnswering := known.answersNow()
	known.WentSilentAt = d.now()
	d.peers[peer] = known

	return wasAnswering, true
}

func (d *Directory) reportPeersKnown(ctx context.Context) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	amountOfAnsweringPeers := 0
	for _, peer := range d.peers {
		if peer.answersNow() {
			amountOfAnsweringPeers++
		}
	}
	d.observer.PeersKnown(ctx, len(d.peers), amountOfAnsweringPeers, d.limits.Capacity)
}

func addressesOf(seed yacymodel.Seed) []string {
	port, ok := seed.Port.Get()
	if !ok {
		return nil
	}

	var hosts []yacymodel.Host
	if primary, ok := seed.PrimaryAddress.Get(); ok {
		hosts = append(hosts, primary)
	}
	if additional, ok := seed.AdditionalAddresses.Get(); ok {
		hosts = append(hosts, additional...)
	}

	addresses := make([]string, 0, len(hosts))
	for _, host := range hosts {
		addresses = append(addresses, "http://"+host.String()+":"+port.String())
	}

	return addresses
}

func addressesLedBy(answeredAddress string, seeded []string) []string {
	if answeredAddress == "" {
		return seeded
	}
	led := make([]string, 0, len(seeded)+1)
	led = append(led, answeredAddress)
	for _, address := range seeded {
		if address != answeredAddress {
			led = append(led, address)
		}
	}

	return led
}

type DirectoryObservers []DirectoryObserver

func (observers DirectoryObservers) PeerAdmitted(
	ctx context.Context,
	peer yacymodel.Hash,
	addresses int,
) {
	for _, observer := range observers {
		observer.PeerAdmitted(ctx, peer, addresses)
	}
}

func (observers DirectoryObservers) PeerAnswered(
	ctx context.Context,
	peer yacymodel.Hash,
	address string,
	answeredAt time.Time,
) {
	for _, observer := range observers {
		observer.PeerAnswered(ctx, peer, address, answeredAt)
	}
}

func (observers DirectoryObservers) PeerWentSilent(ctx context.Context, peer yacymodel.Hash) {
	for _, observer := range observers {
		observer.PeerWentSilent(ctx, peer)
	}
}

func (observers DirectoryObservers) PeerDropped(ctx context.Context, peer yacymodel.Hash) {
	for _, observer := range observers {
		observer.PeerDropped(ctx, peer)
	}
}

func (observers DirectoryObservers) PeersKnown(
	ctx context.Context,
	amountOfKnownPeers, amountOfAnsweringPeers, capacity int,
) {
	for _, observer := range observers {
		observer.PeersKnown(ctx, amountOfKnownPeers, amountOfAnsweringPeers, capacity)
	}
}
