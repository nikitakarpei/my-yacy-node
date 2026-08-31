package peerroster

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const announceRoundsBeforeConfirmationStale = 2

var neverReachable time.Time

type rosterEntry struct {
	primaryAddress yacymodel.Host
	port           yacymodel.Port
	lastReachable  time.Time
	lastContacted  time.Time
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
	reachable map[yacymodel.Hash]yacymodel.Seed
}

func (r *roster) Discover(ctx context.Context, seeds ...yacymodel.Seed) {
	for _, seed := range seeds {
		if seed.Hash == r.self {
			continue
		}
		networkAddress, addressable := seed.NetworkAddress()
		if !addressable {
			continue
		}
		if err := r.discoverOne(ctx, seed.Hash, networkAddress); err != nil {
			slog.WarnContext(
				ctx,
				"peer discovery discarded",
				slog.String("peer", seed.Hash.String()),
				slog.Any("error", err),
			)
		}
	}

	r.evictOverflow(ctx)
	r.observer.ObserveKnownPeers(r.peerCount(ctx))
}

func (r *roster) discoverOne(
	ctx context.Context,
	peerHash yacymodel.Hash,
	networkAddress yacymodel.NetworkAddress,
) error {
	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		entry, known, err := r.peers.Get(tx, peerHash)
		if err != nil {
			return fmt.Errorf("read peer: %w", err)
		}
		if !known {
			entry = rosterEntry{lastContacted: r.now()}
		}
		entry.primaryAddress = networkAddress.Host()
		entry.port = networkAddress.Port()
		if err := r.peers.Put(tx, peerHash, entry); err != nil {
			return fmt.Errorf("store peer: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("discover peer: %w", err)
	}

	return nil
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
	if !r.recordReachable(ctx, seed.Hash, networkAddress) {
		return
	}

	admitted, wasReachable := r.admitReachable(seed.Hash, seed)
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
	r.evictOverflow(ctx)
	r.observer.ObserveKnownPeers(r.peerCount(ctx))
}

func (r *roster) admitReachable(
	peer yacymodel.Hash,
	seed yacymodel.Seed,
) (admitted, wasReachable bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, wasReachable = r.reachable[peer]
	admitted = wasReachable || len(r.reachable) < r.reachableCap
	if admitted {
		r.reachable[peer] = seed
	}
	r.observer.ObserveReachablePeers(len(r.reachable))

	return admitted, wasReachable
}

func (r *roster) recordReachable(
	ctx context.Context,
	peerHash yacymodel.Hash,
	networkAddress yacymodel.NetworkAddress,
) bool {
	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		entry, _, err := r.peers.Get(tx, peerHash)
		if err != nil {
			return fmt.Errorf("read peer: %w", err)
		}

		entry.primaryAddress = networkAddress.Host()
		entry.port = networkAddress.Port()
		entry.lastContacted = r.now()
		entry.lastReachable = r.now()
		if err := r.peers.Put(tx, peerHash, entry); err != nil {
			return fmt.Errorf("store peer: %w", err)
		}

		return nil
	}); err != nil {
		slog.WarnContext(
			ctx,
			"peer contact not recorded",
			slog.String("peer", peerHash.String()),
			slog.Any("error", err),
		)

		return false
	}

	return true
}

func (r *roster) evictReachable(peer yacymodel.Hash) (wasReachable bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, wasReachable = r.reachable[peer]
	delete(r.reachable, peer)
	r.observer.ObserveReachablePeers(len(r.reachable))

	return wasReachable
}

func (r *roster) ConfirmUnreachable(ctx context.Context, peer yacymodel.Hash) {
	if r.evictReachable(peer) {
		slog.DebugContext(ctx, "peer became unreachable", slog.String("peer", peer.String()))
	}

	r.recordUnreachable(ctx, peer)
}

func (r *roster) recordUnreachable(ctx context.Context, peer yacymodel.Hash) {
	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		entry, known, err := r.peers.Get(tx, peer)
		if err != nil {
			return fmt.Errorf("read peer: %w", err)
		}
		if !known {
			return nil
		}

		entry.lastContacted = r.now()
		entry.lastReachable = neverReachable
		if err := r.peers.Put(tx, peer, entry); err != nil {
			return fmt.Errorf("store peer: %w", err)
		}

		return nil
	}); err != nil {
		slog.WarnContext(
			ctx,
			"peer contact not recorded",
			slog.String("peer", peer.String()),
			slog.Any("error", err),
		)
	}
}

func (r *roster) ReachablePeers(_ context.Context) []yacymodel.Seed {
	r.mu.Lock()
	defer r.mu.Unlock()

	peers := make([]yacymodel.Seed, 0, len(r.reachable))
	for _, seed := range r.reachable {
		peers = append(peers, seed)
	}

	return peers
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
			return fmt.Errorf("read peer: %w", err)
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
	knownPeers := r.selectUnreachable(ctx, r.reachableKeys(), limit, func(a, b rosterEntry) bool {
		if !a.lastReachable.Equal(b.lastReachable) {
			return a.lastReachable.After(b.lastReachable)
		}

		return a.lastContacted.Before(b.lastContacted)
	})

	peerHashes := make([]yacymodel.Hash, len(knownPeers))
	for index, knownPeer := range knownPeers {
		peerHashes[index] = knownPeer.peerHash
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
			return fmt.Errorf("read peer: %w", err)
		}
		if !known {
			return nil
		}

		networkAddress, err = yacymodel.NetworkAddressOf(entry.primaryAddress, entry.port)
		if err != nil {
			return fmt.Errorf("network address of peer: %w", err)
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

func (r *roster) reachableKeys() map[yacymodel.Hash]struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	keys := make(map[yacymodel.Hash]struct{}, len(r.reachable))
	for hash := range r.reachable {
		keys[hash] = struct{}{}
	}

	return keys
}

func (r *roster) evictOverflow(ctx context.Context) {
	for _, hash := range r.stalestBeyondCapacity(ctx) {
		if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
			if _, err := r.peers.Delete(tx, hash); err != nil {
				return fmt.Errorf("delete peer: %w", err)
			}

			return nil
		}); err != nil {
			slog.WarnContext(
				ctx,
				"peer eviction failed",
				slog.String("peer", hash.String()),
				slog.Any("error", err),
			)
		}
	}
}

func (r *roster) stalestBeyondCapacity(ctx context.Context) []yacymodel.Hash {
	excess := r.peerCount(ctx) - r.reservoirCap
	if excess <= 0 {
		return nil
	}

	stalePeers := r.stalestUnreachable(ctx, r.reachableKeys(), excess)
	victims := make([]yacymodel.Hash, 0, len(stalePeers))
	for _, stalePeer := range stalePeers {
		victims = append(victims, stalePeer.peerHash)
	}

	return victims
}

func (r *roster) stalestUnreachable(
	ctx context.Context,
	reachable map[yacymodel.Hash]struct{},
	limit int,
) []knownPeer {
	return r.selectUnreachable(ctx, reachable, limit, func(a, b rosterEntry) bool {
		return a.lastContacted.Before(b.lastContacted)
	})
}

func (r *roster) selectUnreachable(
	ctx context.Context,
	reachable map[yacymodel.Hash]struct{},
	limit int,
	precedes func(a, b rosterEntry) bool,
) []knownPeer {
	if limit <= 0 {
		return nil
	}

	keptPeers := make([]knownPeer, 0, limit)
	if err := r.vault.View(ctx, func(tx *vault.Txn) error {
		return r.peers.Scan(
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
		)
	}); err != nil {
		slog.WarnContext(ctx, "peer roster scan failed", slog.Any("error", err))

		return nil
	}

	return keptPeers
}

func (r *roster) peerCount(ctx context.Context) int {
	total := 0
	if err := r.vault.View(ctx, func(tx *vault.Txn) error {
		count, err := r.peers.Len(tx)
		if err != nil {
			return fmt.Errorf("count peers: %w", err)
		}
		total = count

		return nil
	}); err != nil {
		slog.WarnContext(ctx, "peer roster count failed", slog.Any("error", err))

		return 0
	}

	return total
}
