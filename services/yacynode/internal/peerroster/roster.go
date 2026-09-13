package peerroster

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const announceRoundsBeforeConfirmationStale = 2

const msgContactNotRecorded = "peer contact not recorded"

var neverReachable time.Time

type rosterEntry struct {
	primaryAddress yacymodel.Host
	port           yacymodel.Port
	lastReachable  time.Time
	lastContacted  time.Time
}

type reachablePeer struct {
	seed        yacymodel.Seed
	confirmedAt time.Time
}

type knownPeer struct {
	peerHash    yacymodel.Hash
	rosterEntry rosterEntry
}

type roster struct {
	vault            *vault.Vault
	peers            *vault.Collection[yacymodel.Hash, rosterEntry]
	now              func() time.Time
	reservoirCap     int
	reachableCap     int
	announceInterval time.Duration
	self             yacymodel.Hash
	observer         RosterObserver

	mu        sync.Mutex
	reachable map[yacymodel.Hash]reachablePeer
}

func (r *roster) Discover(ctx context.Context, seeds ...yacymodel.Seed) {
	known := 0
	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		if err := r.discoverEach(tx, seeds); err != nil {
			return err
		}
		if err := r.evictOverflow(tx); err != nil {
			return err
		}
		count, err := r.peerCount(tx)
		known = count

		return err
	}); err != nil {
		slog.WarnContext(ctx, "peer discovery discarded", slog.Any("error", err))

		return
	}

	r.observer.ObserveKnownPeers(known)
}

func (r *roster) discoverEach(tx *vault.Txn, seeds []yacymodel.Seed) error {
	for _, seed := range seeds {
		if seed.Hash == r.self {
			continue
		}
		networkAddress, addressable := seed.NetworkAddress()
		if !addressable {
			continue
		}
		if err := r.discoverOne(tx, seed.Hash, networkAddress); err != nil {
			return err
		}
	}

	return nil
}

func (r *roster) discoverOne(
	tx *vault.Txn,
	peerHash yacymodel.Hash,
	networkAddress yacymodel.NetworkAddress,
) error {
	entry, known, err := r.peers.Get(tx, peerHash)
	if err != nil {
		return fmt.Errorf("read peer %s: %w", peerHash, err)
	}
	if !known {
		entry = rosterEntry{lastContacted: r.now()}
	}
	entry.primaryAddress = networkAddress.Host()
	entry.port = networkAddress.Port()
	if err := r.peers.Put(tx, peerHash, entry); err != nil {
		return fmt.Errorf("store peer %s: %w", peerHash, err)
	}

	return nil
}

func (r *roster) evictOverflow(tx *vault.Txn) error {
	victims, err := r.stalestBeyondCapacity(tx)
	if err != nil {
		return err
	}
	for _, hash := range victims {
		if _, err := r.peers.Delete(tx, hash); err != nil {
			return fmt.Errorf("delete peer %s: %w", hash, err)
		}
	}

	return nil
}

func (r *roster) stalestBeyondCapacity(tx *vault.Txn) ([]yacymodel.Hash, error) {
	known, err := r.peerCount(tx)
	if err != nil {
		return nil, err
	}
	excess := known - r.reservoirCap
	if excess <= 0 {
		return nil, nil
	}

	stalePeers, err := r.stalestUnreachable(tx, r.reachableKeys(), excess)
	if err != nil {
		return nil, err
	}
	victims := make([]yacymodel.Hash, 0, len(stalePeers))
	for _, stalePeer := range stalePeers {
		victims = append(victims, stalePeer.peerHash)
	}

	return victims, nil
}

func (r *roster) peerCount(tx *vault.Txn) (int, error) {
	count, err := r.peers.Len(tx)
	if err != nil {
		return 0, fmt.Errorf("count peers: %w", err)
	}

	return count, nil
}

func (r *roster) reachablePeerCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.reachable)
}

func (r *roster) reachableKeys() map[yacymodel.Hash]struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	keys := make(map[yacymodel.Hash]struct{}, len(r.reachable))
	for hash := range r.reachable {
		keys[hash] = struct{}{}
	}

	return keys
}

func (r *roster) stalestUnreachable(
	tx *vault.Txn,
	reachable map[yacymodel.Hash]struct{},
	limit int,
) ([]knownPeer, error) {
	return r.selectUnreachable(tx, reachable, limit, func(a, b rosterEntry) bool {
		return a.lastContacted.Before(b.lastContacted)
	})
}

func (r *roster) selectUnreachable(
	tx *vault.Txn,
	reachable map[yacymodel.Hash]struct{},
	limit int,
	precedes func(a, b rosterEntry) bool,
) ([]knownPeer, error) {
	if limit <= 0 {
		return nil, nil
	}

	keptPeers := make([]knownPeer, 0, limit)
	if err := r.peers.Scan(
		tx,
		vault.EveryKey(),
		func(peerHash yacymodel.Hash, entry rosterEntry) (bool, error) {
			if _, ok := reachable[peerHash]; ok {
				return true, nil
			}

			pos := 0
			for pos < len(keptPeers) && !precedes(entry, keptPeers[pos].rosterEntry) {
				pos++
			}
			if pos >= limit {
				return true, nil
			}
			if len(keptPeers) < limit {
				keptPeers = append(keptPeers, knownPeer{})
			}
			copy(keptPeers[pos+1:], keptPeers[pos:])
			keptPeers[pos] = knownPeer{peerHash: peerHash, rosterEntry: entry}

			return true, nil
		},
	); err != nil {
		return nil, fmt.Errorf("scan peer roster: %w", err)
	}

	return keptPeers, nil
}

func (r *roster) ConfirmReachable(ctx context.Context, seed yacymodel.Seed) {
	if seed.Hash == r.self {
		return
	}
	networkAddress, addressable := seed.NetworkAddress()
	if !addressable {
		slog.WarnContext(
			ctx,
			"reachable peer seed discarded",
			slog.String("peer", seed.Hash.String()),
		)

		return
	}
	confirmedAt := r.now()
	known := 0
	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		if err := r.recordReachable(tx, seed.Hash, networkAddress, confirmedAt); err != nil {
			return err
		}
		if err := r.evictOverflow(tx); err != nil {
			return err
		}
		count, err := r.peerCount(tx)
		known = count

		return err
	}); err != nil {
		slog.WarnContext(
			ctx,
			msgContactNotRecorded,
			slog.String("peer", seed.Hash.String()),
			slog.Any("error", err),
		)

		return
	}

	admitted, wasReachable := r.admitReachable(seed, confirmedAt)
	switch {
	case admitted && !wasReachable:
		slog.DebugContext(ctx, "peer became reachable", slog.String("peer", seed.Hash.String()))
	case !admitted:
		slog.DebugContext(
			ctx,
			"peer reachable but reachable roster full",
			slog.String("peer", seed.Hash.String()),
		)
	}

	r.observer.ObserveKnownPeers(known)
	r.observer.ObserveReachablePeers(r.reachablePeerCount())
}

func (r *roster) recordReachable(
	tx *vault.Txn,
	peerHash yacymodel.Hash,
	networkAddress yacymodel.NetworkAddress,
	confirmedAt time.Time,
) error {
	entry, _, err := r.peers.Get(tx, peerHash)
	if err != nil {
		return fmt.Errorf("read peer %s: %w", peerHash, err)
	}

	entry.primaryAddress = networkAddress.Host()
	entry.port = networkAddress.Port()
	entry.lastContacted = confirmedAt
	entry.lastReachable = confirmedAt
	if err := r.peers.Put(tx, peerHash, entry); err != nil {
		return fmt.Errorf("store peer %s: %w", peerHash, err)
	}

	return nil
}

func (r *roster) admitReachable(
	seed yacymodel.Seed,
	confirmedAt time.Time,
) (admitted, wasReachable bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, wasReachable = r.reachable[seed.Hash]
	admitted = wasReachable || len(r.reachable) < r.reachableCap
	if admitted {
		r.reachable[seed.Hash] = reachablePeer{seed: seed, confirmedAt: confirmedAt}
	}

	return admitted, wasReachable
}

func (r *roster) ConfirmUnreachable(ctx context.Context, peer yacymodel.Hash) {
	if r.evictReachable(peer) {
		slog.DebugContext(ctx, "peer became unreachable", slog.String("peer", peer.String()))
	}

	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		return r.recordUnreachable(tx, peer)
	}); err != nil {
		slog.WarnContext(
			ctx,
			msgContactNotRecorded,
			slog.String("peer", peer.String()),
			slog.Any("error", err),
		)
	}

	r.observer.ObserveReachablePeers(r.reachablePeerCount())
}

func (r *roster) recordUnreachable(tx *vault.Txn, peer yacymodel.Hash) error {
	entry, known, err := r.peers.Get(tx, peer)
	if err != nil {
		return fmt.Errorf("read peer %s: %w", peer, err)
	}
	if !known {
		return nil
	}

	entry.lastContacted = r.now()
	entry.lastReachable = neverReachable
	if err := r.peers.Put(tx, peer, entry); err != nil {
		return fmt.Errorf("store peer %s: %w", peer, err)
	}

	return nil
}

func (r *roster) evictReachable(peer yacymodel.Hash) (wasReachable bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, wasReachable = r.reachable[peer]
	if wasReachable {
		delete(r.reachable, peer)
	}

	return wasReachable
}

func (r *roster) ReachablePeers(_ context.Context) []yacymodel.Seed {
	r.mu.Lock()
	defer r.mu.Unlock()

	peers := make([]yacymodel.Seed, 0, len(r.reachable))
	for _, reachable := range r.reachable {
		peers = append(peers, reachable.seed)
	}

	return peers
}

func (r *roster) MostRecentlyReachablePeers(
	_ context.Context,
	limit int,
) []yacymodel.Seed {
	if limit <= 0 {
		return nil
	}

	ranked := r.reachablePeersByRecency()
	if limit < len(ranked) {
		ranked = ranked[:limit]
	}

	peers := make([]yacymodel.Seed, len(ranked))
	for index, reachable := range ranked {
		peers[index] = reachable.seed
	}

	return peers
}

func (r *roster) reachablePeersByRecency() []reachablePeer {
	r.mu.Lock()
	defer r.mu.Unlock()

	ranked := make([]reachablePeer, 0, len(r.reachable))
	for _, reachable := range r.reachable {
		ranked = append(ranked, reachable)
	}
	slices.SortFunc(ranked, func(a, b reachablePeer) int {
		return b.confirmedAt.Compare(a.confirmedAt)
	})

	return ranked
}

func (r *roster) IsReachable(_ context.Context, peer yacymodel.Hash) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, reachable := r.reachable[peer]

	return reachable
}

func (r *roster) IsRecentlyReachable(ctx context.Context, peer yacymodel.Hash) bool {
	cutoff := r.now().Add(-announceRoundsBeforeConfirmationStale * r.announceInterval)

	recent := false
	if err := r.vault.View(ctx, func(tx *vault.Txn) error {
		entry, known, err := r.peers.Get(tx, peer)
		if err != nil {
			return fmt.Errorf("read peer %s: %w", peer, err)
		}
		recent = known && entry.lastReachable.After(cutoff)

		return nil
	}); err != nil {
		slog.WarnContext(
			ctx,
			"peer recency not read, peer assumed credible",
			slog.String("peer", peer.String()),
			slog.Any("error", err),
		)

		return true
	}

	return recent
}

func (r *roster) UnreachablePeerHashes(ctx context.Context, limit int) []yacymodel.Hash {
	var stalePeers []knownPeer
	if err := r.vault.View(ctx, func(tx *vault.Txn) error {
		peers, err := r.selectUnreachable(
			tx,
			r.reachableKeys(),
			limit,
			func(a, b rosterEntry) bool {
				if !a.lastReachable.Equal(b.lastReachable) {
					return a.lastReachable.After(b.lastReachable)
				}

				return a.lastContacted.Before(b.lastContacted)
			},
		)
		stalePeers = peers

		return err
	}); err != nil {
		slog.WarnContext(ctx, "peer roster scan failed", slog.Any("error", err))

		return nil
	}

	peerHashes := make([]yacymodel.Hash, len(stalePeers))
	for index, stalePeer := range stalePeers {
		peerHashes[index] = stalePeer.peerHash
	}

	return peerHashes
}

func (r *roster) NetworkAddressOf(
	ctx context.Context,
	peer yacymodel.Hash,
) (yacymodel.NetworkAddress, bool) {
	var networkAddress yacymodel.NetworkAddress
	found := false
	if err := r.vault.View(ctx, func(tx *vault.Txn) error {
		entry, known, err := r.peers.Get(tx, peer)
		if err != nil {
			return fmt.Errorf("read peer %s: %w", peer, err)
		}
		if !known {
			return nil
		}

		networkAddress, err = yacymodel.NetworkAddressOf(entry.primaryAddress, entry.port)
		if err != nil {
			return fmt.Errorf("network address of peer %s: %w", peer, err)
		}
		found = true

		return nil
	}); err != nil {
		slog.WarnContext(
			ctx,
			"peer network address not read",
			slog.String("peer", peer.String()),
			slog.Any("error", err),
		)

		return yacymodel.NetworkAddress{}, false
	}

	return networkAddress, found
}
