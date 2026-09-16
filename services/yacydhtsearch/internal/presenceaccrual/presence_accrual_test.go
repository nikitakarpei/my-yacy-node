package presenceaccrual_test

import (
	"context"
	"testing"
	"time"

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

type silentObserver struct{}

func (silentObserver) PeerAnsweredForTheFirstTime(context.Context, yacymodel.Hash, string) {}
func (silentObserver) PeerEarnedPresence(context.Context, yacymodel.Hash, time.Duration)   {}
func (silentObserver) PeersObserved(context.Context, int)                                  {}

func startOfObservation() time.Time {
	return time.Date(2026, time.September, 15, 12, 0, 0, 0, time.UTC)
}

func peerAtAddress(t *testing.T, symbol byte) probeanswerhistory.PeerAtAddress {
	t.Helper()

	hash, err := yacymodel.ParseHash(string([]byte{
		symbol, symbol, symbol, symbol, symbol, symbol,
		symbol, symbol, symbol, symbol, symbol, symbol,
	}))
	if err != nil {
		t.Fatalf("parse hash: %v", err)
	}

	return probeanswerhistory.PeerAtAddress{Hash: hash, Address: answeringAddress}
}

func answeredAt(
	peer probeanswerhistory.PeerAtAddress,
	at time.Time,
) probeanswerhistory.ProbeAnswer {
	return probeanswerhistory.ProbeAnswer{PeerAtAddress: peer, AnsweredAt: at}
}

func TestTheFirstAnswerOfAPeerEarnsItNoPresence(t *testing.T) {
	t.Parallel()

	presence := presenceaccrual.PresenceAccrualFrom(nil, wideAccrualLimits, silentObserver{})
	peer := peerAtAddress(t, 'a')

	observedPeer, credited := presence.Credit(t.Context(), answeredAt(peer, startOfObservation()))

	if !credited {
		t.Fatal("Credit refused the first answer of a peer")
	}
	if observedPeer.Presence != 0 {
		t.Fatalf("Presence = %v, want none earned by a first answer", observedPeer.Presence)
	}
	if !observedPeer.FirstAnsweredAt.Equal(observedPeer.LatestAnsweredAt) {
		t.Fatalf(
			"observed peer = %+v, want its first and latest answer to be the same",
			observedPeer,
		)
	}
}

func TestAFurtherAnswerEarnsThePresenceSinceTheAnswerBeforeIt(t *testing.T) {
	t.Parallel()

	presence := presenceaccrual.PresenceAccrualFrom(nil, wideAccrualLimits, silentObserver{})
	peer := peerAtAddress(t, 'a')
	presence.Credit(t.Context(), answeredAt(peer, startOfObservation()))

	observedPeer, _ := presence.Credit(t.Context(),
		answeredAt(peer, startOfObservation().Add(time.Minute)),
	)

	if observedPeer.Presence != time.Minute {
		t.Fatalf("Presence = %v, want the minute since the answer before it", observedPeer.Presence)
	}
}

func TestAnAnswerAfterALongSilenceEarnsOnlyTheContinuityLimit(t *testing.T) {
	t.Parallel()

	presence := presenceaccrual.PresenceAccrualFrom(nil, wideAccrualLimits, silentObserver{})
	peer := peerAtAddress(t, 'a')
	presence.Credit(t.Context(), answeredAt(peer, startOfObservation()))

	observedPeer, _ := presence.Credit(t.Context(),
		answeredAt(peer, startOfObservation().Add(24*time.Hour)),
	)

	if observedPeer.Presence != continuityLimit {
		t.Fatalf("Presence = %v, want no more than the continuity limit %v",
			observedPeer.Presence, continuityLimit)
	}
}

func TestAnAnswerOlderThanTheLatestOneIsNotCredited(t *testing.T) {
	t.Parallel()

	presence := presenceaccrual.PresenceAccrualFrom(nil, wideAccrualLimits, silentObserver{})
	peer := peerAtAddress(t, 'a')
	presence.Credit(t.Context(), answeredAt(peer, startOfObservation().Add(time.Minute)))

	_, credited := presence.Credit(t.Context(), answeredAt(peer, startOfObservation()))

	if credited {
		t.Fatal("Credit accepted an answer older than the latest one it holds")
	}
}

func TestPresenceCarriesOnFromTheObservedPeersItStartsWith(t *testing.T) {
	t.Parallel()

	peer := peerAtAddress(t, 'a')
	presence := presenceaccrual.PresenceAccrualFrom(
		[]presenceaccrual.ObservedPeer{{
			PeerAtAddress:    peer,
			FirstAnsweredAt:  startOfObservation(),
			LatestAnsweredAt: startOfObservation(),
			Presence:         time.Hour,
		}},
		wideAccrualLimits,
		silentObserver{},
	)

	observedPeer, _ := presence.Credit(t.Context(),
		answeredAt(peer, startOfObservation().Add(time.Minute)),
	)

	if observedPeer.Presence != time.Hour+time.Minute {
		t.Fatalf("Presence = %v, want the hour it started with and the minute it just earned",
			observedPeer.Presence)
	}
}

func TestPresenceIsHeldForNoMorePeersThanItsCapacity(t *testing.T) {
	t.Parallel()

	presence := presenceaccrual.PresenceAccrualFrom(nil, presenceaccrual.PresenceAccrualLimits{
		Capacity:        1,
		ContinuityLimit: continuityLimit,
	}, silentObserver{})
	first, second := peerAtAddress(t, 'a'), peerAtAddress(t, 'b')
	presence.Credit(t.Context(), answeredAt(first, startOfObservation()))
	presence.Credit(t.Context(), answeredAt(second, startOfObservation()))

	if !presence.LatestAnswerOf(first).IsZero() {
		t.Fatal("the first peer still has a latest answer, want it released at the capacity of 1")
	}
	if presence.LatestAnswerOf(second).IsZero() {
		t.Fatal("the peer that answered last has no latest answer, want the answer it gave")
	}
}

func TestAPeerNeverObservedHasEarnedNoPresence(t *testing.T) {
	t.Parallel()

	presence := presenceaccrual.PresenceAccrualFrom(nil, wideAccrualLimits, silentObserver{})

	if earned := presence.EarnedPresenceOf(peerAtAddress(t, 'a')); earned != 0 {
		t.Fatalf("EarnedPresenceOf = %v, want none for a peer never observed", earned)
	}
	if latest := presence.LatestAnswerOf(peerAtAddress(t, 'a')); !latest.IsZero() {
		t.Fatalf("LatestAnswerOf = %v, want no answer for a peer never observed", latest)
	}
}
