package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	peerpresenceredis "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresence/redis"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	networkName      = "yacydhtsearch"
	answersKeptFor   = 24 * time.Hour
	snapshotsKey     = "yacydhtsearch:peer-presence"
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

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }

func presenceOver(
	t *testing.T,
	shared *miniredis.Miniredis,
	limits presenceaccrual.PresenceAccrualLimits,
	snapshotInterval time.Duration,
) *peerpresenceredis.PeerPresence {
	t.Helper()

	client := goredis.NewClient(&goredis.Options{Addr: shared.Addr()})

	return peerpresenceredis.New(
		probeanswerhistory.New(
			client,
			networkName,
			answersKeptFor,
			wallClock{},
			probeanswerhistory.HistoryObservers{},
		),
		client,
		peerpresenceredis.PeerPresenceLimits{
			AccrualLimits:    limits,
			SnapshotInterval: snapshotInterval,
		},
		silentObserver{},
		silentObserver{},
	)
}

func consuming(
	t *testing.T,
	presence *peerpresenceredis.PeerPresence,
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
	presence *peerpresenceredis.PeerPresence,
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

	presence := presenceOver(t, miniredis.RunT(t), wideAccrualLimits, snapshotSeldom)
	defer consuming(t, presence)()
	peer := hashOf(t, 'a')

	presence.PeerAnswered(t.Context(), peer, answeringAddress, startOfObservation())

	foldedWithin(t, presence, peerAtAddress(peer), peerObserved)
}

func TestAnInstanceEarnsPresenceFromAnswersItDidNotPublish(t *testing.T) {
	t.Parallel()

	shared := miniredis.RunT(t)
	publishing := presenceOver(t, shared, wideAccrualLimits, snapshotSeldom)
	defer consuming(t, publishing)()
	sibling := presenceOver(t, shared, wideAccrualLimits, snapshotSeldom)
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

	shared := miniredis.RunT(t)
	before := presenceOver(t, shared, wideAccrualLimits, snapshotOnEvery)
	stop := consuming(t, before)
	peer := hashOf(t, 'a')
	before.PeerAnswered(t.Context(), peer, answeringAddress, startOfObservation())
	before.PeerAnswered(
		t.Context(), peer, answeringAddress, startOfObservation().Add(time.Minute),
	)
	foldedWithin(t, before, peerAtAddress(peer), presenceEarned(time.Minute))
	snapshottedWithin(t, shared)
	stop()
	dropEveryAnswer(shared)

	after := presenceOver(t, shared, wideAccrualLimits, snapshotOnEvery)
	defer consuming(t, after)()

	foldedWithin(t, after, peerAtAddress(peer), presenceEarned(time.Minute))
}

func snapshottedWithin(t *testing.T, server *miniredis.Miniredis) {
	t.Helper()

	deadline := time.Now().Add(foldingDeadline)
	for {
		if keys, err := server.HKeys(snapshotsKey); err == nil && len(keys) > 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("no peer is snapshotted after %v, want the earned presence kept",
				foldingDeadline)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func dropEveryAnswer(shared *miniredis.Miniredis) {
	shared.Del("yacydhtsearch:probe-answers:" + networkName)
}
