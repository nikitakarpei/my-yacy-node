package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	peerpresencejetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresence/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	networkName      = "yacydhtsearch"
	streamName       = "peers-answered"
	bucketName       = "peer-presence"
	answeringAddress = "http://10.0.0.1:8090"
	continuityLimit  = 10 * time.Minute
	wideCapacity     = 16
	snapshotSeldom   = time.Hour
	snapshotOnEvery  = 0
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
func (silentObserver) SnapshotsReadFailed(context.Context, error)                          {}
func (silentObserver) SnapshotUndecodable(context.Context, string, error)                  {}
func (silentObserver) SnapshotWriteFailed(context.Context, string, error)                  {}
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
		Subjects: []string{probeanswerhistory.SubjectOfEveryProbeAnswerIn(networkName)},
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
	snapshotInterval time.Duration,
) *peerpresencejetstream.PeerPresence {
	t.Helper()

	bucket, err := stream.KeyValue(t.Context(), bucketName)
	if err != nil {
		t.Fatalf("open bucket: %v", err)
	}

	return peerpresencejetstream.New(
		probeanswerhistory.New(
			stream,
			streamName,
			networkName,
			probeanswerhistory.HistoryObservers{},
		),
		bucket,
		peerpresencejetstream.PeerPresenceLimits{
			AccrualLimits:    limits,
			SnapshotInterval: snapshotInterval,
		},
		silentObserver{},
		silentObserver{},
	)
}

func consuming(
	t *testing.T,
	presence *peerpresencejetstream.PeerPresence,
) context.CancelFunc {
	t.Helper()

	consumed, stop := context.WithCancel(t.Context())
	go presence.FoldTheProbeAnswerHistory(consumed)

	return stop
}

func peerAtAddress(peer yacymodel.Hash) probeanswerhistory.PeerAtAddress {
	return probeanswerhistory.PeerAtAddress{Hash: peer, Address: answeringAddress}
}

func foldedWithin(
	t *testing.T,
	presence *peerpresencejetstream.PeerPresence,
	peer probeanswerhistory.PeerAtAddress,
	folded func(earnedPresence time.Duration, latestAnswer time.Time) bool,
) {
	t.Helper()

	deadline := time.Now().Add(foldingDeadline)
	for {
		earnedPresence := presence.EarnedPresenceOf(t.Context(), peer)
		latestAnswer := presence.LatestAnswerOf(t.Context(), peer)
		if folded(earnedPresence, latestAnswer) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("earned presence = %v, latest answer = %v after %v, want the answers folded",
				earnedPresence, latestAnswer, foldingDeadline)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func peerObserved(_ time.Duration, latestAnswer time.Time) bool {
	return !latestAnswer.IsZero()
}

func presenceEarned(earned time.Duration) func(time.Duration, time.Time) bool {
	return func(earnedPresence time.Duration, _ time.Time) bool {
		return earnedPresence == earned
	}
}

func TestAnAnsweringPeerBecomesObservedOnceTheStreamIsFolded(t *testing.T) {
	t.Parallel()

	presence := presenceOver(t, sharedJetStream(t), wideAccrualLimits, snapshotSeldom)
	defer consuming(t, presence)()
	peer := hashOf(t, 'a')

	presence.PeerAnswered(t.Context(), peer, answeringAddress, startOfObservation())

	foldedWithin(t, presence, peerAtAddress(peer), peerObserved)
}

func TestAnInstanceEarnsPresenceFromAnswersItDidNotPublish(t *testing.T) {
	t.Parallel()

	stream := sharedJetStream(t)
	publishing := presenceOver(t, stream, wideAccrualLimits, snapshotSeldom)
	defer consuming(t, publishing)()
	sibling := presenceOver(t, stream, wideAccrualLimits, snapshotSeldom)
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
	before := presenceOver(t, stream, wideAccrualLimits, snapshotOnEvery)
	stop := consuming(t, before)
	peer := hashOf(t, 'a')
	before.PeerAnswered(t.Context(), peer, answeringAddress, startOfObservation())
	before.PeerAnswered(
		t.Context(), peer, answeringAddress, startOfObservation().Add(time.Minute),
	)
	foldedWithin(t, before, peerAtAddress(peer), presenceEarned(time.Minute))
	snapshottedWithin(t, stream)
	stop()
	dropEveryAnswer(t, stream)

	after := presenceOver(t, stream, wideAccrualLimits, snapshotOnEvery)
	defer consuming(t, after)()

	foldedWithin(t, after, peerAtAddress(peer), presenceEarned(time.Minute))
}

func snapshottedWithin(t *testing.T, stream natsjetstream.JetStream) {
	t.Helper()

	bucket, err := stream.KeyValue(t.Context(), bucketName)
	if err != nil {
		t.Fatalf("open bucket: %v", err)
	}
	deadline := time.Now().Add(foldingDeadline)
	for {
		if keys, err := bucket.Keys(t.Context()); err == nil && len(keys) > 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("no peer is snapshotted after %v, want the earned presence kept",
				foldingDeadline)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func dropEveryAnswer(t *testing.T, stream natsjetstream.JetStream) {
	t.Helper()

	answers, err := stream.Stream(t.Context(), streamName)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	if err := answers.Purge(t.Context()); err != nil {
		t.Fatalf("drop the answers: %v", err)
	}
}
