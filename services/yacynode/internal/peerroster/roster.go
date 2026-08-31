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

const msgContactNotRecorded = "peer contact not recorded"

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
		if _, reachable := seed.NetworkAddress(); !reachable {
			continue
		}
		if err := r.discoverOne(tx, seed); err != nil {
			return err
		}
	}

	return nil
}

func (r *roster) discoverOne(tx *vault.Txn, seed yacymodel.Seed) error {
	entry, known, err := r.peers.Get(tx, seed.Hash)
	if err != nil {
		return fmt.Errorf("read peer %s: %w", seed.Hash, err)
	}
	if !known {
		entry = rosterEntry{lastContacted: r.now()}
	}
	entry.seed = seed
	if err := r.peers.Put(tx, seed.Hash, entry); err != nil {
		return fmt.Errorf("store peer %s: %w", seed.Hash, err)
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

	stale, err := r.stalestUnreachable(tx, r.reachableKeys(), excess)
	if err != nil {
		return nil, err
	}
	victims := make([]yacymodel.Hash, 0, len(stale))
	for _, entry := range stale {
		victims = append(victims, entry.seed.Hash)
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
) ([]rosterEntry, error) {
	return r.selectUnreachable(tx, reachable, limit, func(a, b rosterEntry) bool {
		return a.lastContacted.Before(b.lastContacted)
	})
}

func (r *roster) selectUnreachable(
	tx *vault.Txn,
	reachable map[yacymodel.Hash]struct{},
	limit int,
	precedes func(a, b rosterEntry) bool,
) ([]rosterEntry, error) {
	if limit <= 0 {
		return nil, nil
	}

	kept := make([]rosterEntry, 0, limit)
	if err := r.peers.Scan(
		tx,
		vault.EveryKey(),
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
	); err != nil {
		return nil, fmt.Errorf("scan peer roster: %w", err)
	}

	return kept, nil
}

func (r *roster) ConfirmReachable(ctx context.Context, peer yacymodel.Hash) {
	var (
		confirmed yacymodel.Seed
		found     bool
	)
	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		seed, known, err := r.recordContact(tx, peer, r.now())
		confirmed, found = seed, known

		return err
	}); err != nil {
		slog.WarnContext(
			ctx,
			msgContactNotRecorded,
			slog.String("peer", peer.String()),
			slog.Any("error", err),
		)

		return
	}
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

func (r *roster) recordContact(
	tx *vault.Txn,
	peer yacymodel.Hash,
	lastReachable time.Time,
) (yacymodel.Seed, bool, error) {
	entry, known, err := r.peers.Get(tx, peer)
	if err != nil {
		return yacymodel.Seed{}, false, fmt.Errorf("read peer %s: %w", peer, err)
	}
	if !known {
		return yacymodel.Seed{}, false, nil
	}

	entry.lastContacted = r.now()
	entry.lastReachable = lastReachable
	if err := r.peers.Put(tx, peer, entry); err != nil {
		return yacymodel.Seed{}, false, fmt.Errorf("store peer %s: %w", peer, err)
	}

	return entry.seed, true, nil
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

func (r *roster) ConfirmUnreachable(ctx context.Context, peer yacymodel.Hash) {
	if r.evictReachable(peer) {
		slog.DebugContext(ctx, "peer became unreachable", slog.String("peer", peer.String()))
	}

	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		_, _, err := r.recordContact(tx, peer, neverReachable)

		return err
	}); err != nil {
		slog.WarnContext(
			ctx,
			msgContactNotRecorded,
			slog.String("peer", peer.String()),
			slog.Any("error", err),
		)
	}
}

func (r *roster) evictReachable(peer yacymodel.Hash) (wasReachable bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, wasReachable = r.reachable[peer]
	delete(r.reachable, peer)
	r.observer.ObserveReachablePeers(len(r.reachable))

	return wasReachable
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

func (r *roster) UnreachablePeers(ctx context.Context, limit int) []yacymodel.Seed {
	var stale []rosterEntry
	if err := r.vault.View(ctx, func(tx *vault.Txn) error {
		entries, err := r.selectUnreachable(
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
		stale = entries

		return err
	}); err != nil {
		slog.WarnContext(ctx, "peer roster scan failed", slog.Any("error", err))

		return nil
	}

	peers := make([]yacymodel.Seed, len(stale))
	for i, entry := range stale {
		peers[i] = entry.seed
	}

	return peers
}
