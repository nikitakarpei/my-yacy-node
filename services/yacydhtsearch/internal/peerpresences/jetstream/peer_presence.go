// Package jetstream keeps every answer this deployment observes in a NATS
// stream that all of its instances share, and folds that stream into the
// presence each peer has earned. An instance credits nothing it observes
// directly: it publishes the answer, and its own answers reach it the same way
// its siblings' answers do, so every instance folds the same answers in the
// same order and holds the same presence. A snapshot in a key-value bucket
// carries the presence one peer has earned and the stream sequence that
// presence includes, so an instance that starts again reads every snapshot and
// folds the stream from the oldest sequence any of them is missing. Answers
// behind that sequence are in every snapshot already and are purged.
package jetstream

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync/atomic"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerAnsweredStream struct {
	JetStream   natsjetstream.JetStream
	Name        string
	NetworkName string
}

func SubjectOfEveryPeerAnsweredIn(networkName string) string {
	return networkName + ".*.*"
}

type snapshottedPeer struct {
	presenceaccrual.ObservedPeer
	FoldedUpTo uint64
}

type PeerPresence struct {
	peerAnsweredStream     PeerAnsweredStream
	snapshots              natsjetstream.KeyValue
	accrualLimits          presenceaccrual.PresenceAccrualLimits
	accrual                atomic.Pointer[presenceaccrual.PresenceAccrual]
	changedPeers           map[presenceaccrual.PeerAtAddress]snapshottedPeer
	answersSinceCompaction int
	accrualObserver        presenceaccrual.PresenceAccrualObserver
	presenceObserver       PeerPresenceObserver
}

func New(
	peerAnsweredStream PeerAnsweredStream,
	snapshots natsjetstream.KeyValue,
	accrualLimits presenceaccrual.PresenceAccrualLimits,
	accrualObserver presenceaccrual.PresenceAccrualObserver,
	presenceObserver PeerPresenceObserver,
) *PeerPresence {
	presence := &PeerPresence{
		peerAnsweredStream: peerAnsweredStream,
		snapshots:          snapshots,
		accrualLimits:      accrualLimits,
		changedPeers:       map[presenceaccrual.PeerAtAddress]snapshottedPeer{},
		accrualObserver:    accrualObserver,
		presenceObserver:   presenceObserver,
	}
	presence.accrual.Store(presenceaccrual.PresenceAccrualFrom(nil, accrualLimits, accrualObserver))

	return presence
}

func (h *PeerPresence) ObservedPeerAt(
	_ context.Context,
	peerAtAddress presenceaccrual.PeerAtAddress,
) (presenceaccrual.ObservedPeer, bool) {
	return h.accrual.Load().ObservedPeerAt(peerAtAddress)
}

func (h *PeerPresence) PeerAnswered(
	ctx context.Context,
	peer yacymodel.Hash,
	address string,
	answeredAt time.Time,
) {
	peerAtAddress := presenceaccrual.PeerAtAddress{Hash: peer, Address: address}
	encoded, err := json.Marshal(presenceaccrual.PeerAnswered{
		PeerAtAddress: peerAtAddress,
		AnsweredAt:    answeredAt,
	})
	if err != nil {
		h.presenceObserver.PeerAnsweredPublishFailed(ctx, err)

		return
	}
	_, err = h.peerAnsweredStream.JetStream.Publish(ctx, h.subjectOf(peerAtAddress), encoded)
	if err != nil {
		h.presenceObserver.PeerAnsweredPublishFailed(ctx, err)
	}
}

func (h *PeerPresence) ConsumeThePeerAnsweredStream(ctx context.Context) {
	peersSnapshotted := h.peersSnapshotted(ctx)
	h.accrual.Store(presenceaccrual.PresenceAccrualFrom(
		observedPeersAmong(peersSnapshotted), h.accrualLimits, h.accrualObserver,
	))

	peerAnsweredMessages, err := h.peerAnsweredMessagesFrom(
		ctx, h.sequenceToFoldFrom(ctx, peersSnapshotted),
	)
	if err != nil {
		h.presenceObserver.PeerAnsweredStreamEnded(ctx, err)

		return
	}
	defer context.AfterFunc(ctx, peerAnsweredMessages.Stop)()
	defer peerAnsweredMessages.Stop()

	for {
		peerAnsweredMessage, err := peerAnsweredMessages.Next()
		if err != nil {
			h.presenceObserver.PeerAnsweredStreamEnded(ctx, err)

			return
		}
		h.credit(ctx, peerAnsweredMessage)
	}
}

func (h *PeerPresence) peersSnapshotted(ctx context.Context) []snapshottedPeer {
	watcher, err := h.snapshots.Watch(ctx, h.subjectOfEveryPeer(), natsjetstream.IgnoreDeletes())
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

func observedPeersAmong(peersSnapshotted []snapshottedPeer) []presenceaccrual.ObservedPeer {
	observedPeers := make([]presenceaccrual.ObservedPeer, 0, len(peersSnapshotted))
	for _, peerSnapshotted := range peersSnapshotted {
		observedPeers = append(observedPeers, peerSnapshotted.ObservedPeer)
	}

	return observedPeers
}

func (h *PeerPresence) sequenceToFoldFrom(
	ctx context.Context,
	peersSnapshotted []snapshottedPeer,
) uint64 {
	foldFrom := oldestSequenceMissingFrom(peersSnapshotted)
	firstStreamSequence, streamRead := h.firstStreamSequence(ctx)
	if streamRead && firstStreamSequence > foldFrom {
		return firstStreamSequence
	}

	return foldFrom
}

func oldestSequenceMissingFrom(peersSnapshotted []snapshottedPeer) uint64 {
	if len(peersSnapshotted) == 0 {
		return 0
	}
	foldedUpTo := peersSnapshotted[0].FoldedUpTo
	for _, peerSnapshotted := range peersSnapshotted {
		foldedUpTo = min(foldedUpTo, peerSnapshotted.FoldedUpTo)
	}

	return foldedUpTo + 1
}

func (h *PeerPresence) firstStreamSequence(ctx context.Context) (uint64, bool) {
	stream, err := h.peerAnsweredStream.JetStream.Stream(ctx, h.peerAnsweredStream.Name)
	if err != nil {
		h.presenceObserver.PeerAnsweredStreamEnded(ctx, err)

		return 0, false
	}
	streamState, err := stream.Info(ctx)
	if err != nil {
		h.presenceObserver.PeerAnsweredStreamEnded(ctx, err)

		return 0, false
	}

	return streamState.State.FirstSeq, true
}

func (h *PeerPresence) peerAnsweredMessagesFrom(
	ctx context.Context,
	foldFrom uint64,
) (natsjetstream.MessagesContext, error) {
	consumer, err := h.peerAnsweredStream.JetStream.OrderedConsumer(
		ctx,
		h.peerAnsweredStream.Name,
		natsjetstream.OrderedConsumerConfig{
			FilterSubjects: []string{h.subjectOfEveryPeer()},
			DeliverPolicy:  natsjetstream.DeliverByStartSequencePolicy,
			OptStartSeq:    foldFrom,
		},
	)
	if err != nil {
		return nil, err //nolint:wrapcheck // the caller reports it to the stream observer
	}

	return consumer.Messages() //nolint:wrapcheck // the caller reports it to the stream observer
}

func (h *PeerPresence) credit(ctx context.Context, peerAnsweredMessage natsjetstream.Msg) {
	delivered, err := peerAnsweredMessage.Metadata()
	if err != nil {
		h.presenceObserver.PeerAnsweredMessageUndecodable(ctx, 0, err)

		return
	}
	var peerAnswered presenceaccrual.PeerAnswered
	if err := json.Unmarshal(peerAnsweredMessage.Data(), &peerAnswered); err != nil {
		h.presenceObserver.PeerAnsweredMessageUndecodable(ctx, delivered.Sequence.Stream, err)

		return
	}
	if observedPeer, credited := h.accrual.Load().Credit(ctx, peerAnswered); credited {
		h.changedPeers[observedPeer.PeerAtAddress] = snapshottedPeer{
			ObservedPeer: observedPeer,
			FoldedUpTo:   delivered.Sequence.Stream,
		}
		h.reportCredited(ctx, observedPeer)
	}
	h.compactWhenDue(ctx)
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

func (h *PeerPresence) compactWhenDue(ctx context.Context) {
	h.answersSinceCompaction++
	if h.answersSinceCompaction < h.accrualLimits.Capacity {
		return
	}
	h.answersSinceCompaction = 0
	h.compact(ctx)
}

func (h *PeerPresence) compact(ctx context.Context) {
	amountOfSnapshots := len(h.changedPeers)
	for peerAtAddress, peerSnapshotted := range h.changedPeers {
		if !h.snapshot(ctx, peerAtAddress, peerSnapshotted) {
			return
		}
	}
	clear(h.changedPeers)
	h.presenceObserver.PeersSnapshotted(ctx, amountOfSnapshots)

	purgedUpTo := oldestSequenceMissingFrom(h.peersSnapshotted(ctx))
	if purgedUpTo == 0 {
		return
	}
	stream, err := h.peerAnsweredStream.JetStream.Stream(ctx, h.peerAnsweredStream.Name)
	if err != nil {
		h.presenceObserver.PeerAnsweredStreamPurgeFailed(ctx, err)

		return
	}
	if err := stream.Purge(ctx, natsjetstream.WithPurgeSequence(purgedUpTo)); err != nil {
		h.presenceObserver.PeerAnsweredStreamPurgeFailed(ctx, err)

		return
	}
	h.presenceObserver.PeerAnsweredStreamPurged(ctx, purgedUpTo)
}

func (h *PeerPresence) snapshot(
	ctx context.Context,
	peerAtAddress presenceaccrual.PeerAtAddress,
	peerSnapshotted snapshottedPeer,
) bool {
	key := h.subjectOf(peerAtAddress)
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

func (h *PeerPresence) PeerAdmitted(context.Context, yacymodel.Hash, int) {}

func (h *PeerPresence) PeerWentSilent(context.Context, yacymodel.Hash) {}

func (h *PeerPresence) PeerDropped(context.Context, yacymodel.Hash) {}

func (h *PeerPresence) PeersKnown(context.Context, int, int, int) {}

func (h *PeerPresence) subjectOf(peerAtAddress presenceaccrual.PeerAtAddress) string {
	return h.peerAnsweredStream.NetworkName + "." + peerAtAddress.Hash.String() + "." +
		base64.RawURLEncoding.EncodeToString([]byte(peerAtAddress.Address))
}

func (h *PeerPresence) subjectOfEveryPeer() string {
	return SubjectOfEveryPeerAnsweredIn(h.peerAnsweredStream.NetworkName)
}
