// Package jetstream folds the probe answer history into the presence each peer
// has earned. An instance credits nothing it observes directly: it appends the
// answer to the history, and its own answers reach it the same way its
// siblings' answers do, so every instance folds the same answers in the same
// order and holds the same presence. A snapshot in a key-value bucket carries
// the presence one peer has earned and the position in the history that
// presence includes, so an instance that starts again reads every snapshot and
// folds on from the earliest position any of them is missing.
package jetstream

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync/atomic"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type snapshottedPeer struct {
	presenceaccrual.ObservedPeer
	FoldedUpTo probeanswerhistory.ProbeAnswerPosition
}

type PeerPresenceLimits struct {
	AccrualLimits    presenceaccrual.PresenceAccrualLimits
	SnapshotInterval time.Duration
}

type PeerPresence struct {
	answers          probeanswerhistory.History
	snapshots        natsjetstream.KeyValue
	accrualLimits    presenceaccrual.PresenceAccrualLimits
	accrual          atomic.Pointer[presenceaccrual.PresenceAccrual]
	changedPeers     map[probeanswerhistory.PeerAtAddress]snapshottedPeer
	snapshotInterval time.Duration
	snapshotDueAt    time.Time
	accrualObserver  presenceaccrual.PresenceAccrualObserver
	presenceObserver PeerPresenceObserver
}

func New(
	answers probeanswerhistory.History,
	snapshots natsjetstream.KeyValue,
	limits PeerPresenceLimits,
	accrualObserver presenceaccrual.PresenceAccrualObserver,
	presenceObserver PeerPresenceObserver,
) *PeerPresence {
	presence := &PeerPresence{
		answers:          answers,
		snapshots:        snapshots,
		accrualLimits:    limits.AccrualLimits,
		changedPeers:     map[probeanswerhistory.PeerAtAddress]snapshottedPeer{},
		snapshotInterval: limits.SnapshotInterval,
		snapshotDueAt:    time.Now().Add(limits.SnapshotInterval),
		accrualObserver:  accrualObserver,
		presenceObserver: presenceObserver,
	}
	presence.accrual.Store(
		presenceaccrual.PresenceAccrualFrom(nil, limits.AccrualLimits, accrualObserver),
	)

	return presence
}

func (h *PeerPresence) EarnedPresenceOf(
	_ context.Context,
	peerAtAddress probeanswerhistory.PeerAtAddress,
) time.Duration {
	return h.accrual.Load().EarnedPresenceOf(peerAtAddress)
}

func (h *PeerPresence) LatestAnswerOf(
	_ context.Context,
	peerAtAddress probeanswerhistory.PeerAtAddress,
) time.Time {
	return h.accrual.Load().LatestAnswerOf(peerAtAddress)
}

func (h *PeerPresence) PeerAnswered(
	ctx context.Context,
	peer yacymodel.Hash,
	address string,
	answeredAt time.Time,
) {
	h.answers.Append(ctx, probeanswerhistory.ProbeAnswer{
		PeerAtAddress: probeanswerhistory.PeerAtAddress{Hash: peer, Address: address},
		AnsweredAt:    answeredAt,
	})
}

func (h *PeerPresence) FoldTheProbeAnswerHistory(ctx context.Context) {
	peersSnapshotted := h.peersSnapshotted(ctx)
	h.accrual.Store(presenceaccrual.PresenceAccrualFrom(
		observedPeersAmong(peersSnapshotted), h.accrualLimits, h.accrualObserver,
	))

	for position, answer := range h.answers.AnswersAfter(ctx, foldedUpToBy(peersSnapshotted)) {
		h.credit(ctx, position, answer)
		h.snapshotWhenDue(ctx)
	}
}

func (h *PeerPresence) peersSnapshotted(ctx context.Context) []snapshottedPeer {
	watcher, err := h.snapshots.Watch(ctx, keyOfEveryPeer, natsjetstream.IgnoreDeletes())
	if err != nil {
		h.presenceObserver.SnapshotsReadFailed(ctx, err)

		return nil
	}
	defer func() { _ = watcher.Stop() }()

	var peersSnapshotted []snapshottedPeer
	for entry := range watcher.Updates() {
		if entry == nil {
			break
		}
		var peerSnapshotted snapshottedPeer
		if err := json.Unmarshal(entry.Value(), &peerSnapshotted); err != nil {
			h.presenceObserver.SnapshotUndecodable(ctx, entry.Key(), err)

			continue
		}
		peersSnapshotted = append(peersSnapshotted, peerSnapshotted)
	}

	return peersSnapshotted
}

const keyOfEveryPeer = "*.*"

func observedPeersAmong(peersSnapshotted []snapshottedPeer) []presenceaccrual.ObservedPeer {
	observedPeers := make([]presenceaccrual.ObservedPeer, 0, len(peersSnapshotted))
	for _, peerSnapshotted := range peersSnapshotted {
		observedPeers = append(observedPeers, peerSnapshotted.ObservedPeer)
	}

	return observedPeers
}

func foldedUpToBy(peersSnapshotted []snapshottedPeer) probeanswerhistory.ProbeAnswerPosition {
	foldedUpTo := make([]probeanswerhistory.ProbeAnswerPosition, 0, len(peersSnapshotted))
	for _, peerSnapshotted := range peersSnapshotted {
		foldedUpTo = append(foldedUpTo, peerSnapshotted.FoldedUpTo)
	}

	return probeanswerhistory.EarliestOf(foldedUpTo...)
}

func (h *PeerPresence) credit(
	ctx context.Context,
	position probeanswerhistory.ProbeAnswerPosition,
	answer probeanswerhistory.ProbeAnswer,
) {
	if observedPeer, credited := h.accrual.Load().Credit(ctx, answer); credited {
		h.changedPeers[observedPeer.PeerAtAddress] = snapshottedPeer{
			ObservedPeer: observedPeer,
			FoldedUpTo:   position,
		}
		h.reportCredited(ctx, observedPeer)
	}
}

func (h *PeerPresence) reportCredited(
	ctx context.Context,
	observedPeer presenceaccrual.ObservedPeer,
) {
	if observedPeer.LatestAnsweredAt.Equal(observedPeer.FirstAnsweredAt) {
		h.accrualObserver.PeerAnsweredForTheFirstTime(
			ctx, observedPeer.Hash, observedPeer.Address,
		)

		return
	}
	h.accrualObserver.PeerEarnedPresence(ctx, observedPeer.Hash, observedPeer.Presence)
}

func (h *PeerPresence) snapshotWhenDue(ctx context.Context) {
	if time.Now().Before(h.snapshotDueAt) {
		return
	}
	h.snapshotDueAt = time.Now().Add(h.snapshotInterval)
	h.snapshotChangedPeers(ctx)
}

func (h *PeerPresence) snapshotChangedPeers(ctx context.Context) {
	amountOfSnapshots := len(h.changedPeers)
	for peerAtAddress, peerSnapshotted := range h.changedPeers {
		if !h.snapshot(ctx, peerAtAddress, peerSnapshotted) {
			return
		}
	}
	clear(h.changedPeers)
	h.presenceObserver.PeersSnapshotted(ctx, amountOfSnapshots)
}

func (h *PeerPresence) snapshot(
	ctx context.Context,
	peerAtAddress probeanswerhistory.PeerAtAddress,
	peerSnapshotted snapshottedPeer,
) bool {
	key := keyOf(peerAtAddress)
	encoded, err := json.Marshal(peerSnapshotted)
	if err != nil {
		h.presenceObserver.SnapshotWriteFailed(ctx, key, err)

		return false
	}
	if _, err := h.snapshots.Put(ctx, key, encoded); err != nil {
		h.presenceObserver.SnapshotWriteFailed(ctx, key, err)

		return false
	}

	return true
}

func keyOf(peerAtAddress probeanswerhistory.PeerAtAddress) string {
	return peerAtAddress.Hash.String() + "." +
		base64.RawURLEncoding.EncodeToString([]byte(peerAtAddress.Address))
}

func (h *PeerPresence) PeerAdmitted(context.Context, yacymodel.Hash, int) {}

func (h *PeerPresence) PeerWentSilent(context.Context, yacymodel.Hash) {}

func (h *PeerPresence) PeerDropped(context.Context, yacymodel.Hash) {}

func (h *PeerPresence) PeersKnown(context.Context, int, int, int) {}
