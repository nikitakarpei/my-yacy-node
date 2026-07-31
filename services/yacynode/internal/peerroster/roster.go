package peerroster

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const peersBucket vault.Name = "peerroster"

var demotedLastSeen = time.Unix(0, 0)

type roster struct {
	vault        *vault.Vault
	peers        *vault.Collection[rosterEntry]
	now          func() time.Time
	reservoirCap int
	activeCap    int
	self         yacymodel.Hash
	observer     RosterObserver

	mu     sync.Mutex
	active map[yacymodel.Hash]yacymodel.Seed
}

func (r *roster) key(hash yacymodel.Hash) vault.Key {
	return vault.Key(hash.String())
}

func (r *roster) Discover(ctx context.Context, seeds ...yacymodel.Seed) {
	for _, seed := range seeds {
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
	key := r.key(seed.Hash)
	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		_, known, err := r.peers.Get(tx, key)
		if err != nil {
			return fmt.Errorf("read peer: %w", err)
		}
		if known {
			return nil
		}
		if err := r.peers.Put(tx, key, rosterEntry{seed: seed, lastSeen: r.now()}); err != nil {
			return fmt.Errorf("store peer: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("discover peer: %w", err)
	}

	return nil
}

func (r *roster) ConfirmReachable(ctx context.Context, peer yacymodel.Hash) {
	confirmed, found := r.touch(ctx, peer)
	if !found {
		return
	}

	if admitted, wasActive := r.admitActive(peer, confirmed); admitted && !wasActive {
		slog.DebugContext(ctx, "peer became reachable", slog.String("peer", peer.String()))
	}
}

func (r *roster) admitActive(peer yacymodel.Hash, seed yacymodel.Seed) (admitted, wasActive bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, wasActive = r.active[peer]
	admitted = wasActive || len(r.active) < r.activeCap
	if admitted {
		r.active[peer] = seed
	}
	r.observer.ObserveReachablePeers(len(r.active))

	return admitted, wasActive
}

func (r *roster) touch(ctx context.Context, peer yacymodel.Hash) (yacymodel.Seed, bool) {
	var confirmed yacymodel.Seed
	found := false
	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		entry, known, err := r.peers.Get(tx, r.key(peer))
		if err != nil {
			return fmt.Errorf("read peer: %w", err)
		}
		if !known {
			return nil
		}

		entry.lastSeen = r.now()
		if err := r.peers.Put(tx, r.key(peer), entry); err != nil {
			return fmt.Errorf("store peer: %w", err)
		}
		confirmed, found = entry.seed, true

		return nil
	}); err != nil {
		slog.WarnContext(
			ctx,
			"peer reachability not recorded",
			slog.String("peer", peer.String()),
			slog.Any("error", err),
		)

		return yacymodel.Seed{}, false
	}

	return confirmed, found
}

func (r *roster) evictActive(peer yacymodel.Hash) (wasActive bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, wasActive = r.active[peer]
	delete(r.active, peer)
	r.observer.ObserveReachablePeers(len(r.active))

	return wasActive
}

func (r *roster) ConfirmUnreachable(ctx context.Context, peer yacymodel.Hash) {
	if r.evictActive(peer) {
		slog.DebugContext(ctx, "peer became unreachable", slog.String("peer", peer.String()))
	}

	if err := r.demote(ctx, peer); err != nil {
		slog.WarnContext(
			ctx,
			"peer demotion failed",
			slog.String("peer", peer.String()),
			slog.Any("error", err),
		)
	}
}

func (r *roster) demote(ctx context.Context, peer yacymodel.Hash) error {
	return r.vault.Update(ctx, func(tx *vault.Txn) error {
		entry, known, err := r.peers.Get(tx, r.key(peer))
		if err != nil {
			return fmt.Errorf("read peer: %w", err)
		}
		if !known {
			return nil
		}

		entry.lastSeen = demotedLastSeen
		if err := r.peers.Put(tx, r.key(peer), entry); err != nil {
			return fmt.Errorf("store peer: %w", err)
		}

		return nil
	})
}

func (r *roster) PeersResponsibleFor(
	ctx context.Context,
	position yacymodel.DHTPosition,
	want int,
) []yacymodel.Seed {
	if want <= 0 {
		return nil
	}

	type candidate struct {
		seed     yacymodel.Seed
		distance yacymodel.DHTPosition
	}

	candidates := make([]candidate, 0, len(r.ReachablePeers(ctx)))
	for _, seed := range r.ReachablePeers(ctx) {
		if seed.Hash == r.self {
			continue
		}
		capabilities, ok := seed.Capabilities.Get()
		if !ok || !capabilities.AcceptRemoteIndex {
			continue
		}

		peerPos, err := yacymodel.WordPosition(seed.Hash)
		if err != nil {
			slog.WarnContext(
				ctx,
				"peer position not computed",
				slog.String("peer", seed.Hash.String()),
				slog.Any("error", err),
			)

			continue
		}

		candidates = append(
			candidates,
			candidate{seed: seed, distance: yacymodel.Distance(position, peerPos)},
		)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].distance < candidates[j].distance
	})

	if len(candidates) > want {
		candidates = candidates[:want]
	}

	targets := make([]yacymodel.Seed, len(candidates))
	for i, c := range candidates {
		targets[i] = c.seed
	}

	return targets
}

func (r *roster) ReachablePeers(_ context.Context) []yacymodel.Seed {
	r.mu.Lock()
	defer r.mu.Unlock()

	peers := make([]yacymodel.Seed, 0, len(r.active))
	for _, seed := range r.active {
		peers = append(peers, seed)
	}

	return peers
}

// Future: a recency index would replace this scan with a bounded range read.
func (r *roster) FreshestPeers(ctx context.Context, limit int) []yacymodel.Seed {
	targets, active := r.activeSnapshot()

	need := limit - len(targets)
	if need <= 0 {
		if len(targets) > limit {
			targets = targets[:limit]
		}

		return targets
	}

	for _, entry := range r.freshestInactive(ctx, active, need) {
		targets = append(targets, entry.seed)
	}

	return targets
}

func (r *roster) activeSnapshot() ([]yacymodel.Seed, map[yacymodel.Hash]struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	seeds := make([]yacymodel.Seed, 0, len(r.active))
	keys := make(map[yacymodel.Hash]struct{}, len(r.active))
	for hash, seed := range r.active {
		seeds = append(seeds, seed)
		keys[hash] = struct{}{}
	}

	return seeds, keys
}

func (r *roster) evictOverflow(ctx context.Context) {
	for _, hash := range r.stalestBeyondCapacity(ctx) {
		if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
			if _, err := r.peers.Delete(tx, r.key(hash)); err != nil {
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
	_, active := r.activeSnapshot()

	excess := r.peerCount(ctx) - r.reservoirCap
	if excess <= 0 {
		return nil
	}

	stale := r.stalestInactive(ctx, active, excess)
	victims := make([]yacymodel.Hash, 0, len(stale))
	for _, entry := range stale {
		victims = append(victims, entry.seed.Hash)
	}

	return victims
}

func (r *roster) freshestInactive(
	ctx context.Context,
	active map[yacymodel.Hash]struct{},
	limit int,
) []rosterEntry {
	return r.selectInactive(ctx, active, limit, func(a, b rosterEntry) bool {
		return a.lastSeen.Compare(b.lastSeen) > 0
	})
}

func (r *roster) stalestInactive(
	ctx context.Context,
	active map[yacymodel.Hash]struct{},
	limit int,
) []rosterEntry {
	return r.selectInactive(ctx, active, limit, func(a, b rosterEntry) bool {
		return a.lastSeen.Compare(b.lastSeen) < 0
	})
}

func (r *roster) selectInactive(
	ctx context.Context,
	active map[yacymodel.Hash]struct{},
	limit int,
	precedes func(a, b rosterEntry) bool,
) []rosterEntry {
	if limit <= 0 {
		return nil
	}

	kept := make([]rosterEntry, 0, limit)
	if err := r.vault.View(ctx, func(tx *vault.Txn) error {
		return r.peers.Scan(tx, nil, func(_ vault.Key, entry rosterEntry) (bool, error) {
			if _, ok := active[entry.seed.Hash]; ok {
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
		})
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
