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

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresence"
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
	peerpresence.ObservedPeer
	FoldedUpTo uint64
}

type PeerHistory struct {
	peerAnsweredStream     PeerAnsweredStream
	snapshots              natsjetstream.KeyValue
	presenceLimits         peerpresence.PeerPresenceLimits
	presence               atomic.Pointer[peerpresence.PeerPresence]
	changedPeers           map[peerpresence.PeerAtAddress]snapshottedPeer
	answersSinceCompaction int
	presenceObserver       peerpresence.PeerPresenceObserver
	historyObserver        PeerHistoryObserver
}

func New(
	peerAnsweredStream PeerAnsweredStream,
	snapshots natsjetstream.KeyValue,
	presenceLimits peerpresence.PeerPresenceLimits,
	presenceObserver peerpresence.PeerPresenceObserver,
	historyObserver PeerHistoryObserver,
) *PeerHistory {
	history := &PeerHistory{
		peerAnsweredStream: peerAnsweredStream,
		snapshots:          snapshots,
		presenceLimits:     presenceLimits,
		changedPeers:       map[peerpresence.PeerAtAddress]snapshottedPeer{},
		presenceObserver:   presenceObserver,
		historyObserver:    historyObserver,
	}
	history.presence.Store(peerpresence.PeerPresenceFrom(nil, presenceLimits, presenceObserver))

	return history
}

func (h *PeerHistory) ObservedPeers(context.Context) []peerpresence.ObservedPeer {
	return h.presence.Load().ObservedPeers()
}

func (h *PeerHistory) PeerAnswered(
	ctx context.Context,
	peer yacymodel.Hash,
	address string,
	answeredAt time.Time,
) {
	peerAtAddress := peerpresence.PeerAtAddress{Hash: peer, Address: address}
	encoded, err := json.Marshal(peerpresence.PeerAnswered{
		PeerAtAddress: peerAtAddress,
		AnsweredAt:    answeredAt,
	})
	if err != nil {
		h.historyObserver.PeerAnsweredPublishFailed(ctx, err)

		return
	}
	_, err = h.peerAnsweredStream.JetStream.Publish(ctx, h.subjectOf(peerAtAddress), encoded)
	if err != nil {
		h.historyObserver.PeerAnsweredPublishFailed(ctx, err)
	}
}

func (h *PeerHistory) ConsumeThePeerAnsweredStream(ctx context.Context) {
	peersSnapshotted := h.peersSnapshotted(ctx)
	h.presence.Store(peerpresence.PeerPresenceFrom(
		observedPeersAmong(peersSnapshotted), h.presenceLimits, h.presenceObserver,
	))

	peerAnsweredMessages, err := h.peerAnsweredMessagesFrom(
		ctx, h.sequenceToFoldFrom(ctx, peersSnapshotted),
	)
	if err != nil {
		h.historyObserver.PeerAnsweredStreamEnded(ctx, err)

		return
	}
	defer context.AfterFunc(ctx, peerAnsweredMessages.Stop)()
	defer peerAnsweredMessages.Stop()

	for {
		peerAnsweredMessage, err := peerAnsweredMessages.Next()
		if err != nil {
			h.historyObserver.PeerAnsweredStreamEnded(ctx, err)

			return
		}
		h.credit(ctx, peerAnsweredMessage)
	}
}

func (h *PeerHistory) peersSnapshotted(ctx context.Context) []snapshottedPeer {
	watcher, err := h.snapshots.Watch(ctx, h.subjectOfEveryPeer(), natsjetstream.IgnoreDeletes())
	if err != nil {
		h.historyObserver.SnapshotsReadFailed(ctx, err)

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
			h.historyObserver.SnapshotUndecodable(ctx, entry.Key(), err)

			continue
		}
		peersSnapshotted = append(peersSnapshotted, peerSnapshotted)
	}

	return peersSnapshotted
}

func observedPeersAmong(peersSnapshotted []snapshottedPeer) []peerpresence.ObservedPeer {
	observedPeers := make([]peerpresence.ObservedPeer, 0, len(peersSnapshotted))
	for _, peerSnapshotted := range peersSnapshotted {
		observedPeers = append(observedPeers, peerSnapshotted.ObservedPeer)
	}

	return observedPeers
}

func (h *PeerHistory) sequenceToFoldFrom(
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

func (h *PeerHistory) firstStreamSequence(ctx context.Context) (uint64, bool) {
	stream, err := h.peerAnsweredStream.JetStream.Stream(ctx, h.peerAnsweredStream.Name)
	if err != nil {
		h.historyObserver.PeerAnsweredStreamEnded(ctx, err)

		return 0, false
	}
	streamState, err := stream.Info(ctx)
	if err != nil {
		h.historyObserver.PeerAnsweredStreamEnded(ctx, err)

		return 0, false
	}

	return streamState.State.FirstSeq, true
}

func (h *PeerHistory) peerAnsweredMessagesFrom(
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

func (h *PeerHistory) credit(ctx context.Context, peerAnsweredMessage natsjetstream.Msg) {
	delivered, err := peerAnsweredMessage.Metadata()
	if err != nil {
		h.historyObserver.PeerAnsweredMessageUndecodable(ctx, 0, err)

		return
	}
	var peerAnswered peerpresence.PeerAnswered
	if err := json.Unmarshal(peerAnsweredMessage.Data(), &peerAnswered); err != nil {
		h.historyObserver.PeerAnsweredMessageUndecodable(ctx, delivered.Sequence.Stream, err)

		return
	}
	if observedPeer, credited := h.presence.Load().Credit(ctx, peerAnswered); credited {
		h.changedPeers[observedPeer.PeerAtAddress] = snapshottedPeer{
			ObservedPeer: observedPeer,
			FoldedUpTo:   delivered.Sequence.Stream,
		}
		h.reportCredited(ctx, observedPeer)
	}
	h.compactWhenDue(ctx)
}

func (h *PeerHistory) reportCredited(
	ctx context.Context,
	observedPeer peerpresence.ObservedPeer,
) {
	if observedPeer.LatestAnsweredAt.Equal(observedPeer.FirstAnsweredAt) {
		h.presenceObserver.PeerAnsweredForTheFirstTime(
			ctx, observedPeer.Hash, observedPeer.Address,
		)

		return
	}
	h.presenceObserver.PeerEarnedPresence(ctx, observedPeer.Hash, observedPeer.Presence)
}

func (h *PeerHistory) compactWhenDue(ctx context.Context) {
	h.answersSinceCompaction++
	if h.answersSinceCompaction < h.presenceLimits.Capacity {
		return
	}
	h.answersSinceCompaction = 0
	h.compact(ctx)
}

func (h *PeerHistory) compact(ctx context.Context) {
	amountOfSnapshots := len(h.changedPeers)
	for peerAtAddress, peerSnapshotted := range h.changedPeers {
		if !h.snapshot(ctx, peerAtAddress, peerSnapshotted) {
			return
		}
	}
	clear(h.changedPeers)
	h.historyObserver.PeersSnapshotted(ctx, amountOfSnapshots)

	purgedUpTo := oldestSequenceMissingFrom(h.peersSnapshotted(ctx))
	if purgedUpTo == 0 {
		return
	}
	stream, err := h.peerAnsweredStream.JetStream.Stream(ctx, h.peerAnsweredStream.Name)
	if err != nil {
		h.historyObserver.PeerAnsweredStreamPurgeFailed(ctx, err)

		return
	}
	if err := stream.Purge(ctx, natsjetstream.WithPurgeSequence(purgedUpTo)); err != nil {
		h.historyObserver.PeerAnsweredStreamPurgeFailed(ctx, err)

		return
	}
	h.historyObserver.PeerAnsweredStreamPurged(ctx, purgedUpTo)
}

func (h *PeerHistory) snapshot(
	ctx context.Context,
	peerAtAddress peerpresence.PeerAtAddress,
	peerSnapshotted snapshottedPeer,
) bool {
	key := h.subjectOf(peerAtAddress)
	encoded, err := json.Marshal(peerSnapshotted)
	if err != nil {
		h.historyObserver.SnapshotWriteFailed(ctx, key, err)

		return false
	}
	if _, err := h.snapshots.Put(ctx, key, encoded); err != nil {
		h.historyObserver.SnapshotWriteFailed(ctx, key, err)

		return false
	}

	return true
}

func (h *PeerHistory) PeerAdmitted(context.Context, yacymodel.Hash, int) {}

func (h *PeerHistory) PeerWentSilent(context.Context, yacymodel.Hash) {}

func (h *PeerHistory) PeerDropped(context.Context, yacymodel.Hash) {}

func (h *PeerHistory) PeersKnown(context.Context, int, int, int) {}

func (h *PeerHistory) subjectOf(peerAtAddress peerpresence.PeerAtAddress) string {
	return h.peerAnsweredStream.NetworkName + "." + peerAtAddress.Hash.String() + "." +
		base64.RawURLEncoding.EncodeToString([]byte(peerAtAddress.Address))
}

func (h *PeerHistory) subjectOfEveryPeer() string {
	return SubjectOfEveryPeerAnsweredIn(h.peerAnsweredStream.NetworkName)
}
