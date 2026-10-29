package peerroster

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

const announceRoundsBeforeConfirmationStale = 2

const msgContactNotRecorded = "peer contact not recorded"

var neverReachable time.Time

type rosterEntry struct {
	addresses           yacymodel.PeerAddresses
	binding             yacymodel.Optional[yacymodel.VerifiedAddressSignature]
	lastAnsweredBaseURL yacymodel.Optional[peerwire.BaseURL]
	lastReachable       time.Time
	lastContacted       time.Time
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
	announceInterval time.Duration
	self             yacymodel.Hash
	networkName      string
	observer         RosterObserver
	reachable        *reachablePeers
}

func (r *roster) Discover(ctx context.Context, seeds ...yacymodel.Seed) {
	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		return r.discoverEach(tx, seeds)
	}); err != nil {
		slog.WarnContext(ctx, "peer discovery discarded", slog.Any("error", err))

		return
	}

	r.keepWithinCapacity(ctx)
}

func (r *roster) discoverEach(tx *vault.Txn, seeds []yacymodel.Seed) error {
	for _, seed := range seeds {
		if seed.Hash == r.self {
			continue
		}
		addresses, addressable := seed.Addresses()
		if !addressable {
			continue
		}
		if err := r.discoverOne(tx, seed, addresses); err != nil {
			return err
		}
	}

	return nil
}

func (r *roster) discoverOne(
	tx *vault.Txn,
	seed yacymodel.Seed,
	addresses yacymodel.PeerAddresses,
) error {
	known, isKnown, err := r.peers.Get(tx, seed.Hash)
	if err != nil {
		return fmt.Errorf("read peer %s: %w", seed.Hash, err)
	}
	now := r.now()
	verifiedSignature := seed.VerifiedAddressSignatureIn(r.networkName)
	standing := standingOf(verifiedSignature, known, now)
	switch {
	case standing == futureSeed:
		return nil
	case !isKnown:
		return r.storePeer(tx, seed.Hash, rosterEntry{
			addresses:     addresses,
			binding:       verifiedSignature,
			lastContacted: now,
		})
	case standing == newerSeed:
		return r.storePeer(tx, seed.Hash, known.movedTo(addresses, verifiedSignature))
	default:
		return nil
	}
}

func (r *roster) storePeer(tx *vault.Txn, peerHash yacymodel.Hash, entry rosterEntry) error {
	previous, wasKnown, err := r.peers.PutReturning(tx, peerHash, entry)
	if err != nil {
		return fmt.Errorf("store peer %s: %w", peerHash, err)
	}
	if wasKnown && entry.movedFrom(previous) {
		tx.RunAfterCommit(func() { r.reachable.evict(peerHash) })
	}

	return nil
}

func (e rosterEntry) movedFrom(previous rosterEntry) bool {
	return !sameBinding(e.binding, previous.binding) ||
		!sameAddresses(e.addresses, previous.addresses)
}

func sameBinding(
	first yacymodel.Optional[yacymodel.VerifiedAddressSignature],
	second yacymodel.Optional[yacymodel.VerifiedAddressSignature],
) bool {
	firstSignature, firstBound := first.Get()
	secondSignature, secondBound := second.Get()

	return firstBound == secondBound &&
		firstSignature.Identity == secondSignature.Identity &&
		firstSignature.IssuedAt.Equal(secondSignature.IssuedAt)
}

func sameAddresses(first, second yacymodel.PeerAddresses) bool {
	return first.Port == second.Port &&
		first.SecurePort == second.SecurePort &&
		maps.Equal(hostSetFrom(first.Hosts), hostSetFrom(second.Hosts))
}

func hostSetFrom(hosts []yacymodel.Host) map[yacymodel.Host]struct{} {
	hostSet := make(map[yacymodel.Host]struct{}, len(hosts))
	for _, host := range hosts {
		hostSet[host] = struct{}{}
	}

	return hostSet
}

func (e rosterEntry) movedTo(
	addresses yacymodel.PeerAddresses,
	binding yacymodel.Optional[yacymodel.VerifiedAddressSignature],
) rosterEntry {
	e.addresses = addresses
	e.binding = binding

	return e
}

func (r *roster) keepWithinCapacity(ctx context.Context) {
	known, err := r.knownPeerCount(ctx)
	if err == nil && known > r.reservoirCap {
		err = r.evictOverflow(ctx)
		if err == nil {
			known, err = r.knownPeerCount(ctx)
		}
	}
	if err != nil {
		slog.WarnContext(ctx, "peer roster kept above its capacity", slog.Any("error", err))

		return
	}

	r.observer.ObserveKnownPeers(known)
}

func (r *roster) knownPeerCount(ctx context.Context) (int, error) {
	known := 0
	err := r.vault.View(ctx, func(tx *vault.Txn) error {
		count, err := r.peerCount(tx)
		known = count

		return err
	})

	return known, err
}

func (r *roster) evictOverflow(ctx context.Context) error {
	return r.vault.Update(ctx, func(tx *vault.Txn) error {
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
	})
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

	stalePeers, err := r.stalestUnreachable(tx, r.reachable.hashSet(), excess)
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

func (r *roster) ConfirmReachable(
	ctx context.Context,
	seed yacymodel.Seed,
	answeredBaseURL peerwire.BaseURL,
) {
	confirmedAt := r.now()
	standing, recorded := r.recordReachable(ctx, seed, answeredBaseURL, confirmedAt)
	switch {
	case !recorded:
	case standing == currentSeed:
		r.reachable.refresh(seed.Hash, confirmedAt)
	default:
		r.admitReachable(ctx, seed, confirmedAt)
	}
}

func (r *roster) recordReachable(
	ctx context.Context,
	seed yacymodel.Seed,
	answeredBaseURL peerwire.BaseURL,
	confirmedAt time.Time,
) (seedStanding, bool) {
	if seed.Hash == r.self {
		return legacySeed, false
	}
	addresses, addressable := seed.Addresses()
	if !addressable {
		slog.WarnContext(
			ctx,
			"reachable peer seed discarded",
			slog.String("peer", seed.Hash.String()),
		)

		return legacySeed, false
	}
	contact := rosterEntry{
		addresses:           addresses,
		binding:             seed.VerifiedAddressSignatureIn(r.networkName),
		lastAnsweredBaseURL: yacymodel.Some(answeredBaseURL),
		lastReachable:       confirmedAt,
		lastContacted:       confirmedAt,
	}
	var standing seedStanding
	if err := r.vault.Update(ctx, func(tx *vault.Txn) error {
		var err error
		standing, err = r.recordReachableIn(tx, seed.Hash, contact)

		return err
	}); err != nil {
		slog.WarnContext(
			ctx,
			msgContactNotRecorded,
			slog.String("peer", seed.Hash.String()),
			slog.Any("error", err),
		)

		return standing, false
	}
	if reason, dropped := dropReasonOf(standing).Get(); dropped {
		r.reportDroppedSeed(ctx, seed.Hash, reason)

		return standing, false
	}

	return standing, true
}

func (r *roster) recordReachableIn(
	tx *vault.Txn,
	peerHash yacymodel.Hash,
	contact rosterEntry,
) (seedStanding, error) {
	known, _, err := r.peers.Get(tx, peerHash)
	if err != nil {
		return legacySeed, fmt.Errorf("read peer %s: %w", peerHash, err)
	}
	standing := standingOf(contact.binding, known, r.now())
	if dropReasonOf(standing).Present() {
		return standing, nil
	}

	return standing, r.storePeer(tx, peerHash, contact)
}

func (r *roster) reportDroppedSeed(
	ctx context.Context,
	peerHash yacymodel.Hash,
	reason SeedDropReason,
) {
	slog.WarnContext(
		ctx,
		"peer seed dropped",
		slog.String("peer", peerHash.String()),
		slog.String("reason", string(reason)),
	)
	r.observer.ObserveDroppedSeed(reason)
}

func (r *roster) admitReachable(ctx context.Context, seed yacymodel.Seed, confirmedAt time.Time) {
	admitted, wasReachable := r.reachable.admit(seed, confirmedAt)
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

	r.keepWithinCapacity(ctx)
	r.observer.ObserveReachablePeers(r.reachable.amount())
}

func (r *roster) ConfirmUnreachable(ctx context.Context, peer yacymodel.Hash) {
	if r.reachable.evict(peer) {
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

	r.observer.ObserveReachablePeers(r.reachable.amount())
}

func (r *roster) recordUnreachable(tx *vault.Txn, peer yacymodel.Hash) error {
	entry, known, err := r.peers.Get(tx, peer)
	if err != nil {
		return fmt.Errorf("read peer %s: %w", peer, err)
	}
	if !known {
		return nil
	}

	entry.lastAnsweredBaseURL = yacymodel.None[peerwire.BaseURL]()
	entry.lastContacted = r.now()
	entry.lastReachable = neverReachable

	return r.storePeer(tx, peer, entry)
}

func (r *roster) ReachablePeers(_ context.Context) []yacymodel.Seed {
	return r.reachable.seeds()
}

func (r *roster) MostRecentlyReachablePeers(
	_ context.Context,
	limit int,
) []yacymodel.Seed {
	return r.reachable.mostRecentSeeds(limit)
}

func (r *roster) IsReachable(_ context.Context, peer yacymodel.Hash) bool {
	return r.reachable.contains(peer)
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
			r.reachable.hashSet(),
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

func (r *roster) RecordOf(ctx context.Context, peer yacymodel.Hash) (PeerRecord, bool) {
	entry, known := r.entryOf(ctx, peer)

	return entry.recordOf(peer), known
}

func (e rosterEntry) recordOf(peer yacymodel.Hash) PeerRecord {
	callee := e.calleeFrom(peer)

	return PeerRecord{
		Callee:          callee,
		Addresses:       e.addresses,
		AnsweredBaseURL: e.answeredBaseURLAmong(callee.BaseURLsFrom(e.addresses)),
	}
}

func (e rosterEntry) calleeFrom(peer yacymodel.Hash) peerwire.Callee {
	binding, bound := e.binding.Get()
	if !bound {
		return peerwire.UnidentifiedCallee(peer)
	}

	return peerwire.IdentifiedCallee(binding.Identity)
}

func (e rosterEntry) answeredBaseURLAmong(
	baseURLs []peerwire.BaseURL,
) yacymodel.Optional[peerwire.BaseURL] {
	answered, isAnswered := e.lastAnsweredBaseURL.Get()
	if !isAnswered || !slices.Contains(baseURLs, answered) {
		return yacymodel.None[peerwire.BaseURL]()
	}

	return e.lastAnsweredBaseURL
}

func (r *roster) entryOf(ctx context.Context, peer yacymodel.Hash) (rosterEntry, bool) {
	var entry rosterEntry
	known := false
	if err := r.vault.View(ctx, func(tx *vault.Txn) error {
		var err error
		entry, known, err = r.peers.Get(tx, peer)
		if err != nil {
			return fmt.Errorf("read peer %s: %w", peer, err)
		}

		return nil
	}); err != nil {
		slog.WarnContext(
			ctx,
			"peer not read from roster",
			slog.String("peer", peer.String()),
			slog.Any("error", err),
		)

		return rosterEntry{}, false
	}

	return entry, known
}
