package memory_test

import (
	"context"
	"testing"
	"time"

	peerpresencememory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresence/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
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

func TestAnAnsweringPeerBecomesObserved(t *testing.T) {
	t.Parallel()

	reported := &reportedPresence{}
	presence := peerpresencememory.New(wideAccrualLimits, reported)
	peer := hashOf(t, 'a')

	presence.PeerAnswered(t.Context(), peer, answeringAddress, startOfObservation())

	latestAnswer := presence.LatestAnswerOf(t.Context(), probeanswerhistory.PeerAtAddress{
		Hash:    peer,
		Address: answeringAddress,
	})
	if !latestAnswer.Equal(startOfObservation()) {
		t.Fatalf("LatestAnswerOf = %v, want the answer the peer gave", latestAnswer)
	}
	if reported.firstAnswers != 1 {
		t.Fatalf("reported %+v, want one first answer", reported)
	}
}

func TestAPeerThatKeepsAnsweringEarnsPresence(t *testing.T) {
	t.Parallel()

	reported := &reportedPresence{}
	presence := peerpresencememory.New(wideAccrualLimits, reported)
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

	presence := peerpresencememory.New(wideAccrualLimits, &reportedPresence{})
	peer := hashOf(t, 'a')
	presence.PeerAnswered(t.Context(), peer, answeringAddress, startOfObservation())
	presence.PeerAnswered(
		t.Context(), peer, answeringAddress, startOfObservation().Add(time.Minute),
	)

	presence.PeerDropped(t.Context(), peer)

	earned := presence.EarnedPresenceOf(t.Context(), probeanswerhistory.PeerAtAddress{
		Hash:    peer,
		Address: answeringAddress,
	})
	if earned != time.Minute {
		t.Fatalf("EarnedPresenceOf = %v, want the minute the dropped peer earned", earned)
	}
}
