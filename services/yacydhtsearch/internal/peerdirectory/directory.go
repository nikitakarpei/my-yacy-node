package peerdirectory

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type StalePeerSource interface {
	StalestPeers(ctx context.Context, known []KnownPeer, limit int) []yacymodel.Hash
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

type Directory struct {
	mutex    sync.Mutex
	peers    map[yacymodel.Hash]KnownPeer
	capacity int
	cooldown time.Duration
	now      func() time.Time
	stale    StalePeerSource
	observer DirectoryObserver
}

func New(
	capacity int,
	cooldown time.Duration,
	now func() time.Time,
	stale StalePeerSource,
	observer DirectoryObserver,
) *Directory {
	return &Directory{
		peers:    make(map[yacymodel.Hash]KnownPeer, capacity),
		capacity: capacity,
		cooldown: cooldown,
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

	var admittedPeers []KnownPeer
	var droppedPeers []yacymodel.Hash
	for _, seed := range seeds {
		addresses := addressesOf(seed)
		if len(addresses) == 0 {
			continue
		}
		if known, ok := d.peers[seed.Hash]; ok {
			known.Addresses = addressesLedBy(known.AnsweredAddress, addresses)
			d.peers[seed.Hash] = known
			continue
		}
		roomMade, droppedPeer := d.roomMade(ctx)
		droppedPeers = append(droppedPeers, droppedPeer...)
		if !roomMade {
			continue
		}
		d.peers[seed.Hash] = KnownPeer{
			Hash:       seed.Hash,
			Addresses:  addresses,
			AdmittedAt: d.now(),
		}
		admittedPeers = append(admittedPeers, d.peers[seed.Hash])
	}

	return admittedPeers, droppedPeers
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
		if !peer.answersNow() || d.now().Sub(peer.ChosenAt) < d.cooldown {
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

func (d *Directory) roomMade(ctx context.Context) (bool, []yacymodel.Hash) {
	if len(d.peers) < d.capacity {
		return true, nil
	}
	stalestPeers := d.stale.StalestPeers(ctx, slices.Collect(maps.Values(d.peers)), 1)
	for _, stalePeer := range stalestPeers {
		delete(d.peers, stalePeer)
	}

	return len(d.peers) < d.capacity, stalestPeers
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
	d.observer.PeersKnown(ctx, len(d.peers), amountOfAnsweringPeers, d.capacity)
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
