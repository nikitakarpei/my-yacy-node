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
	StalestPeers(known []KnownPeer, limit int) []yacymodel.Hash
}

type DirectoryObserver interface {
	PeerAdmitted(ctx context.Context, peer yacymodel.Hash, addresses int)
	PeerAnswering(ctx context.Context, peer yacymodel.Hash, address string)
	PeerSilent(ctx context.Context, peer yacymodel.Hash)
	PeerDropped(ctx context.Context, peer yacymodel.Hash)
	DirectoryHolds(ctx context.Context, peers, answeringPeers, capacity int)
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
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for _, seed := range seeds {
		addresses := addressesOf(seed)
		if len(addresses) == 0 {
			continue
		}
		if known, ok := d.peers[seed.Hash]; ok {
			known.Addresses = addresses
			d.peers[seed.Hash] = known
			continue
		}
		if !d.makeRoom(ctx) {
			continue
		}
		d.peers[seed.Hash] = KnownPeer{
			Hash:       seed.Hash,
			Addresses:  addresses,
			AdmittedAt: d.now(),
		}
		d.observer.PeerAdmitted(ctx, seed.Hash, len(addresses))
	}
	d.reportContents(ctx)
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
		if peer.AnsweringAddress == "" || d.now().Sub(peer.ChosenAt) < d.cooldown {
			continue
		}
		askable = append(askable, AskablePeer{Hash: peer.Hash, Address: peer.AnsweringAddress})
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
	d.mutex.Lock()
	defer d.mutex.Unlock()

	known, ok := d.peers[peer]
	if !ok {
		return
	}
	wasSilent := known.AnsweringAddress == ""
	known.AnsweringAddress = address
	known.AnsweredAt = d.now()
	d.peers[peer] = known
	d.observer.PeerAnswering(ctx, peer, address)
	if wasSilent {
		d.reportContents(ctx)
	}
}

func (d *Directory) ConfirmSilent(ctx context.Context, peer yacymodel.Hash) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	known, ok := d.peers[peer]
	if !ok {
		return
	}
	wasAnswering := known.AnsweringAddress != ""
	known.AnsweringAddress = ""
	d.peers[peer] = known
	d.observer.PeerSilent(ctx, peer)
	if wasAnswering {
		d.reportContents(ctx)
	}
}

func (d *Directory) makeRoom(ctx context.Context) bool {
	if len(d.peers) < d.capacity {
		return true
	}
	for _, stale := range d.stale.StalestPeers(slices.Collect(maps.Values(d.peers)), 1) {
		delete(d.peers, stale)
		d.observer.PeerDropped(ctx, stale)
	}

	return len(d.peers) < d.capacity
}

func (d *Directory) reportContents(ctx context.Context) {
	answeringPeers := 0
	for _, peer := range d.peers {
		if peer.AnsweringAddress != "" {
			answeringPeers++
		}
	}
	d.observer.DirectoryHolds(ctx, len(d.peers), answeringPeers, d.capacity)
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

func (observers DirectoryObservers) PeerAnswering(
	ctx context.Context,
	peer yacymodel.Hash,
	address string,
) {
	for _, observer := range observers {
		observer.PeerAnswering(ctx, peer, address)
	}
}

func (observers DirectoryObservers) PeerSilent(ctx context.Context, peer yacymodel.Hash) {
	for _, observer := range observers {
		observer.PeerSilent(ctx, peer)
	}
}

func (observers DirectoryObservers) PeerDropped(ctx context.Context, peer yacymodel.Hash) {
	for _, observer := range observers {
		observer.PeerDropped(ctx, peer)
	}
}

func (observers DirectoryObservers) DirectoryHolds(
	ctx context.Context,
	peers, answeringPeers, capacity int,
) {
	for _, observer := range observers {
		observer.DirectoryHolds(ctx, peers, answeringPeers, capacity)
	}
}
