package memory_test

import (
	"context"
	"testing"
	"time"

	peerpresencesmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresences/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	continuityLimit  = 10 * time.Minute
	wideCapacity     = 16
	answeringAddress = "http://10.0.0.1:8090"
)

var wideAccrualLimits = presenceaccrual.PresenceAccrualLimits{
	Capacity:        wideCapacity,
	ContinuityLimit: continuityLimit,
}

type reportedPresence struct {
	firstAnswers   int
	earnedPresence time.Duration
	observedPeers  int
}

func (r *reportedPresence) PeerAnsweredForTheFirstTime(
	context.Context,
	yacymodel.Hash,
	string,
) {
	r.firstAnswers++
}

func (r *reportedPresence) PeerEarnedPresence(
	_ context.Context,
	_ yacymodel.Hash,
	presence time.Duration,
) {
	r.earnedPresence = presence
}

func (r *reportedPresence) PeersObserved(_ context.Context, amountOfObservedPeers int) {
	r.observedPeers = amountOfObservedPeers
}

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

func TestAnAnsweringPeerBecomesAnObservedPeer(t *testing.T) {
	t.Parallel()

	reported := &reportedPresence{}
	presence := peerpresencesmemory.New(wideAccrualLimits, reported)
	peer := hashOf(t, 'a')

	presence.PeerAnswered(t.Context(), peer, answeringAddress, startOfObservation())

	observed := presence.ObservedPeers(t.Context())
	if len(observed) != 1 || observed[0].Hash != peer {
		t.Fatalf("ObservedPeers = %+v, want the peer that answered", observed)
	}
	if reported.firstAnswers != 1 || reported.observedPeers != 1 {
		t.Fatalf("reported %+v, want one first answer and one observed peer", reported)
	}
}

func TestAPeerThatKeepsAnsweringEarnsPresence(t *testing.T) {
	t.Parallel()

	reported := &reportedPresence{}
	presence := peerpresencesmemory.New(wideAccrualLimits, reported)
	peer := hashOf(t, 'a')

	presence.PeerAnswered(t.Context(), peer, answeringAddress, startOfObservation())
	presence.PeerAnswered(
		t.Context(), peer, answeringAddress, startOfObservation().Add(time.Minute),
	)

	if reported.earnedPresence != time.Minute {
		t.Fatalf("earned presence = %v, want the minute between the answers",
			reported.earnedPresence)
	}
}

func TestAPeerDroppedFromTheDirectoryKeepsThePresenceItEarned(t *testing.T) {
	t.Parallel()

	presence := peerpresencesmemory.New(wideAccrualLimits, &reportedPresence{})
	peer := hashOf(t, 'a')
	presence.PeerAnswered(t.Context(), peer, answeringAddress, startOfObservation())

	presence.PeerDropped(t.Context(), peer)

	if observed := presence.ObservedPeers(t.Context()); len(observed) != 1 {
		t.Fatalf("ObservedPeers = %+v, want the dropped peer to keep what it earned", observed)
	}
}
