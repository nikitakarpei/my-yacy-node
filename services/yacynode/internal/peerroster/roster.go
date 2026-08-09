package peerroster

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

const announceRoundsBeforeConfirmationStale = 2

var neverReachable time.Time

type rosterEntry struct {
	seed          yacymodel.Seed
	lastReachable time.Time
	lastContacted time.Time
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
		if _, reachable := seed.NetworkAddress(); !reachable {
			continue
		}
		if err := r.discoverOne(ctx, seed); err != nil {
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

func (r *roster) discoverOne(ctx context.Context, seed yacymodel.Seed) error {
	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		entry, known, err := r.peers.Get(tx, seed.Hash)
		if err != nil {
			return fmt.Errorf("read peer: %w", err)
		}
		if !known {
			entry = rosterEntry{lastContacted: r.now()}
		}
		entry.seed = seed
		if err := r.peers.Put(tx, seed.Hash, entry); err != nil {
			return fmt.Errorf("store peer: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("discover peer: %w", err)
	}

	return nil
}

func (r *roster) ConfirmReachable(ctx context.Context, peer yacymodel.Hash) {
	confirmed, found := r.recordContact(ctx, peer, true)
	if !found {
		return
	}

	admitted, wasReachable := r.admitReachable(peer, confirmed)
	switch {
	case admitted && !wasReachable:
		slog.DebugContext(ctx, "peer became reachable", slog.String("peer", peer.String()))
	case !admitted:
		slog.DebugContext(
			ctx,
			"peer reachable but reachable roster full",
			slog.String("peer", peer.String()),
		)
	}
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

func (r *roster) recordContact(
	ctx context.Context,
	peer yacymodel.Hash,
	reachable bool,
) (yacymodel.Seed, bool) {
	var confirmed yacymodel.Seed
	found := false
	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		entry, known, err := r.peers.Get(tx, peer)
		if err != nil {
			return fmt.Errorf("read peer: %w", err)
		}
		if !known {
			return nil
		}

		entry.lastContacted = r.now()
		if reachable {
			entry.lastReachable = r.now()
		} else {
			entry.lastReachable = neverReachable
		}
		if err := r.peers.Put(tx, peer, entry); err != nil {
			return fmt.Errorf("store peer: %w", err)
		}
		confirmed, found = entry.seed, true

		return nil
	}); err != nil {
		slog.WarnContext(
			ctx,
			"peer contact not recorded",
			slog.String("peer", peer.String()),
			slog.Any("error", err),
		)

		return yacymodel.Seed{}, false
	}

	return confirmed, found
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

	r.recordContact(ctx, peer, false)
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

func (r *roster) UnreachablePeers(ctx context.Context, limit int) []yacymodel.Seed {
	entries := r.selectUnreachable(ctx, r.reachableKeys(), limit, func(a, b rosterEntry) bool {
		if !a.lastReachable.Equal(b.lastReachable) {
			return a.lastReachable.After(b.lastReachable)
		}

		return a.lastContacted.Before(b.lastContacted)
	})

	peers := make([]yacymodel.Seed, len(entries))
	for i, entry := range entries {
		peers[i] = entry.seed
	}

	return peers
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

	stale := r.stalestUnreachable(ctx, r.reachableKeys(), excess)
	victims := make([]yacymodel.Hash, 0, len(stale))
	for _, entry := range stale {
		victims = append(victims, entry.seed.Hash)
	}

	return victims
}

func (r *roster) stalestUnreachable(
	ctx context.Context,
	reachable map[yacymodel.Hash]struct{},
	limit int,
) []rosterEntry {
	return r.selectUnreachable(ctx, reachable, limit, func(a, b rosterEntry) bool {
		return a.lastContacted.Before(b.lastContacted)
	})
}

func (r *roster) selectUnreachable(
	ctx context.Context,
	reachable map[yacymodel.Hash]struct{},
	limit int,
	precedes func(a, b rosterEntry) bool,
) []rosterEntry {
	if limit <= 0 {
		return nil
	}

	kept := make([]rosterEntry, 0, limit)
	if err := r.vault.View(ctx, func(tx *vault.Txn) error {
		return r.peers.Scan(
			tx,
			vaultkey.EveryKey(),
			func(_ yacymodel.Hash, entry rosterEntry) (bool, error) {
				if _, ok := reachable[entry.seed.Hash]; ok {
					return true, nil
				}

				pos := 0
				for pos < len(kept) && !precedes(entry, kept[pos]) {
					pos++
				}
				if pos >= limit {
					return true, nil
				}
				if len(kept) < limit {
					kept = append(kept, rosterEntry{})
				}
				copy(kept[pos+1:], kept[pos:])
				kept[pos] = entry

				return true, nil
			},
		)
	}); err != nil {
		slog.WarnContext(ctx, "peer roster scan failed", slog.Any("error", err))

		return nil
	}

	return kept
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
