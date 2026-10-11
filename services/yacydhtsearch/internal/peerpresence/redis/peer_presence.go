// Package redis folds the probe answer history into the presence each peer
// has earned. An instance credits nothing it observes directly: it appends the
// answer to the history, and its own answers reach it the same way its
// siblings' answers do, so every instance folds the same answers in the same
// order and holds the same presence. A snapshot on a Redis-protocol server carries
// the presence one peer has earned and the position in the history that
// presence includes, so an instance that starts again reads every snapshot and
// folds on from the earliest position any of them is missing.
package redis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const snapshotsKey = "peer-presence"

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
	client           redis.UniversalClient
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
	client redis.UniversalClient,
	limits PeerPresenceLimits,
	accrualObserver presenceaccrual.PresenceAccrualObserver,
	presenceObserver PeerPresenceObserver,
) *PeerPresence {
	presence := &PeerPresence{
		answers:          answers,
		client:           client,
		accrualLimits:    limits.AccrualLimits,
		changedPeers:     map[probeanswerhistory.PeerAtAddress]snapshottedPeer{},
		snapshotInterval: limits.SnapshotInterval,
		snapshotDueAt:    time.Now().Add(limits.SnapshotInterval),
		accrualObserver:  accrualObserver,
		presenceObserver: presenceObserver,
	}
	presence.accrual.Store(
		presenceaccrual.PresenceAccrualFrom(nil, limits.AccrualLimits),
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
		observedPeersAmong(peersSnapshotted), h.accrualLimits,
	))

	for position, answer := range h.answers.AnswersAfter(ctx, foldedUpToBy(peersSnapshotted)) {
		h.credit(ctx, position, answer)
		h.snapshotWhenDue(ctx)
	}
}

func (h *PeerPresence) peersSnapshotted(ctx context.Context) []snapshottedPeer {
	snapshots, err := h.client.HGetAll(ctx, snapshotsKey).Result()
	if err != nil {
		h.presenceObserver.SnapshotsReadFailed(ctx, err)

		return nil
	}

	peersSnapshotted := make([]snapshottedPeer, 0, len(snapshots))
	for key, encoded := range snapshots {
		var peerSnapshotted snapshottedPeer
		if err := json.Unmarshal([]byte(encoded), &peerSnapshotted); err != nil {
			h.presenceObserver.SnapshotUndecodable(ctx, key, err)

			continue
		}
		peersSnapshotted = append(peersSnapshotted, peerSnapshotted)
	}

	return peersSnapshotted
}

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
	if observedPeer, credited := h.accrual.Load().Credit(answer); credited {
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
	if err := h.client.HSet(ctx, snapshotsKey, key, encoded).Err(); err != nil {
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
