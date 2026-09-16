package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	peerpresencesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresences/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	networkName      = "yacydhtsearch"
	streamName       = "peers-answered"
	bucketName       = "peer-presence"
	answeringAddress = "http://10.0.0.1:8090"
	continuityLimit  = 10 * time.Minute
	wideCapacity     = 16
	foldingDeadline  = 5 * time.Second
)

var wideAccrualLimits = presenceaccrual.PresenceAccrualLimits{
	Capacity:        wideCapacity,
	ContinuityLimit: continuityLimit,
}

type silentObserver struct{}

func (silentObserver) PeerAnsweredForTheFirstTime(context.Context, yacymodel.Hash, string) {}
func (silentObserver) PeerEarnedPresence(context.Context, yacymodel.Hash, time.Duration)   {}
func (silentObserver) PeersObserved(context.Context, int)                                  {}
func (silentObserver) PeerAnsweredPublishFailed(context.Context, error)                    {}
func (silentObserver) PeerAnsweredMessageUndecodable(context.Context, uint64, error)       {}
func (silentObserver) PeerAnsweredStreamEnded(context.Context, error)                      {}
func (silentObserver) SnapshotsReadFailed(context.Context, error)                          {}
func (silentObserver) SnapshotUndecodable(context.Context, string, error)                  {}
func (silentObserver) SnapshotWriteFailed(context.Context, string, error)                  {}
func (silentObserver) PeerAnsweredStreamPurgeFailed(context.Context, error)                {}
func (silentObserver) PeerAnsweredStreamPurged(context.Context, uint64)                    {}
func (silentObserver) PeersSnapshotted(context.Context, int)                               {}

func startOfObservation() time.Time {
	return time.Date(2026, time.September, 15, 12, 0, 0, 0, time.UTC)
}

func hashOf(t *testing.T, symbol byte) yacymodel.Hash {
	t.Helper()

	hash, err := yacymodel.ParseHash(string([]byte{
		symbol, symbol, symbol, symbol, symbol, symbol,
		symbol, symbol, symbol, symbol, symbol, symbol,
	}))
	if err != nil {
		t.Fatalf("parse hash: %v", err)
	}

	return hash
}

func sharedJetStream(t *testing.T) natsjetstream.JetStream {
	t.Helper()

	stream := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	_, err := stream.CreateOrUpdateStream(t.Context(), natsjetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{peerpresencesjetstream.SubjectOfEveryPeerAnsweredIn(networkName)},
	})
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}
	_, err = stream.CreateOrUpdateKeyValue(t.Context(), natsjetstream.KeyValueConfig{
		Bucket: bucketName,
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	return stream
}

func presenceOver(
	t *testing.T,
	stream natsjetstream.JetStream,
	limits presenceaccrual.PresenceAccrualLimits,
) *peerpresencesjetstream.PeerPresence {
	t.Helper()

	bucket, err := stream.KeyValue(t.Context(), bucketName)
	if err != nil {
		t.Fatalf("open bucket: %v", err)
	}

	return peerpresencesjetstream.New(
		peerpresencesjetstream.PeerAnsweredStream{
			JetStream:   stream,
			Name:        streamName,
			NetworkName: networkName,
		},
		bucket,
		limits,
		silentObserver{},
		silentObserver{},
	)
}

func consuming(
	t *testing.T,
	presence *peerpresencesjetstream.PeerPresence,
) context.CancelFunc {
	t.Helper()

	consumed, stop := context.WithCancel(t.Context())
	go presence.ConsumeThePeerAnsweredStream(consumed)

	return stop
}

func peerAtAddress(peer yacymodel.Hash) presenceaccrual.PeerAtAddress {
	return presenceaccrual.PeerAtAddress{Hash: peer, Address: answeringAddress}
}

func foldedWithin(
	t *testing.T,
	presence *peerpresencesjetstream.PeerPresence,
	peer presenceaccrual.PeerAtAddress,
	folded func(presenceaccrual.ObservedPeer, bool) bool,
) {
	t.Helper()

	deadline := time.Now().Add(foldingDeadline)
	for {
		observedPeer, isObserved := presence.ObservedPeerAt(t.Context(), peer)
		if folded(observedPeer, isObserved) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("ObservedPeerAt = %+v after %v, want the answers folded",
				observedPeer, foldingDeadline)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func peerObserved(_ presenceaccrual.ObservedPeer, isObserved bool) bool {
	return isObserved
}

func presenceEarned(earned time.Duration) func(presenceaccrual.ObservedPeer, bool) bool {
	return func(observedPeer presenceaccrual.ObservedPeer, isObserved bool) bool {
		return isObserved && observedPeer.Presence == earned
	}
}

func TestAnAnsweringPeerBecomesObservedOnceTheStreamIsFolded(t *testing.T) {
	t.Parallel()

	presence := presenceOver(t, sharedJetStream(t), wideAccrualLimits)
	defer consuming(t, presence)()
	peer := hashOf(t, 'a')

	presence.PeerAnswered(t.Context(), peer, answeringAddress, startOfObservation())

	foldedWithin(t, presence, peerAtAddress(peer), peerObserved)
}

func TestAnInstanceEarnsPresenceFromAnswersItDidNotPublish(t *testing.T) {
	t.Parallel()

	stream := sharedJetStream(t)
	publishing := presenceOver(t, stream, wideAccrualLimits)
	defer consuming(t, publishing)()
	sibling := presenceOver(t, stream, wideAccrualLimits)
	defer consuming(t, sibling)()
	peer := hashOf(t, 'a')

	publishing.PeerAnswered(t.Context(), peer, answeringAddress, startOfObservation())
	publishing.PeerAnswered(
		t.Context(), peer, answeringAddress, startOfObservation().Add(time.Minute),
	)

	foldedWithin(t, sibling, peerAtAddress(peer), presenceEarned(time.Minute))
}

func TestPresenceSurvivesAnInstanceThatStartsAgainOverTheSameStream(t *testing.T) {
	t.Parallel()

	stream := sharedJetStream(t)
	compactingLimits := presenceaccrual.PresenceAccrualLimits{
		Capacity:        2,
		ContinuityLimit: continuityLimit,
	}
	before := presenceOver(t, stream, compactingLimits)
	stop := consuming(t, before)
	peer := hashOf(t, 'a')
	before.PeerAnswered(t.Context(), peer, answeringAddress, startOfObservation())
	before.PeerAnswered(
		t.Context(), peer, answeringAddress, startOfObservation().Add(time.Minute),
	)
	foldedWithin(t, before, peerAtAddress(peer), presenceEarned(time.Minute))
	purgedWithin(t, stream)
	stop()

	after := presenceOver(t, stream, compactingLimits)
	defer consuming(t, after)()

	foldedWithin(t, after, peerAtAddress(peer), presenceEarned(time.Minute))
}

func purgedWithin(t *testing.T, stream natsjetstream.JetStream) {
	t.Helper()

	deadline := time.Now().Add(foldingDeadline)
	for {
		answers := streamedAnswers(t, stream)
		if answers == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("stream holds %d answers after %v, want them purged behind the snapshots",
				answers, foldingDeadline)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func streamedAnswers(t *testing.T, stream natsjetstream.JetStream) uint64 {
	t.Helper()

	peerAnsweredStream, err := stream.Stream(t.Context(), streamName)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	streamState, err := peerAnsweredStream.Info(t.Context())
	if err != nil {
		t.Fatalf("read stream state: %v", err)
	}

	return streamState.State.Msgs
}
