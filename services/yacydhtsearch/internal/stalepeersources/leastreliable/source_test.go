package leastreliable_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerreliability"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/stalepeersources/leastreliable"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	refreshInterval   = 5 * time.Minute
	longEnoughToProbe = time.Hour
	answeringAddress  = "http://10.0.0.1:8090"
	anotherAddress    = "http://10.0.0.2:8090"
)

var theInstant = time.Unix(1_000_000, 0)

type observedPeers map[presenceaccrual.PeerAtAddress]presenceaccrual.ObservedPeer

func (peers observedPeers) ObservedPeerAt(
	_ context.Context,
	peerAtAddress presenceaccrual.PeerAtAddress,
) (presenceaccrual.ObservedPeer, bool) {
	observedPeer, isObserved := peers[peerAtAddress]

	return observedPeer, isObserved
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

func presenceEarnedAt(
	peer yacymodel.Hash,
	address string,
	presence time.Duration,
) presenceaccrual.ObservedPeer {
	return presenceaccrual.ObservedPeer{
		PeerAtAddress:    presenceaccrual.PeerAtAddress{Hash: peer, Address: address},
		FirstAnsweredAt:  theInstant.Add(-presence),
		LatestAnsweredAt: theInstant,
		Presence:         presence,
	}
}

func presenceOver(observed ...presenceaccrual.ObservedPeer) observedPeers {
	peers := make(observedPeers, len(observed))
	for _, observedPeer := range observed {
		peers[observedPeer.PeerAtAddress] = observedPeer
	}

	return peers
}

func sourceOver(observed observedPeers) leastreliable.Source {
	return leastreliable.New(
		observed,
		peerreliability.ReliabilityWeights{
			MaturationDuration: time.Hour,
			StalenessHorizon:   time.Hour,
		},
		refreshInterval,
		func() time.Time { return theInstant },
	)
}

func knownPeer(peer yacymodel.Hash, addresses ...string) peerdirectory.KnownPeer {
	return peerdirectory.KnownPeer{
		Hash:       peer,
		Addresses:  addresses,
		AdmittedAt: theInstant.Add(-longEnoughToProbe),
	}
}

func TestThePeerWithTheLeastEarnedReliabilityIsTheStalest(t *testing.T) {
	t.Parallel()

	present, transient := hashOf(t, 'a'), hashOf(t, 'b')
	presentPeer := presenceEarnedAt(present, answeringAddress, time.Hour)
	transientPeer := presenceEarnedAt(transient, answeringAddress, time.Minute)
	stalest := sourceOver(presenceOver(presentPeer, transientPeer)).StalestPeers(
		t.Context(),
		[]peerdirectory.KnownPeer{
			knownPeer(present, answeringAddress),
			knownPeer(transient, answeringAddress),
		},
		1,
	)

	if len(stalest) != 1 || stalest[0] != transient {
		t.Fatalf("StalestPeers = %v, want %s", stalest, transient)
	}
}

func TestAPeerSpeaksForItselfThroughTheAddressItHasDoneBestFrom(t *testing.T) {
	t.Parallel()

	moved, transient := hashOf(t, 'a'), hashOf(t, 'b')
	movedPeer := presenceEarnedAt(moved, anotherAddress, time.Hour)
	transientPeer := presenceEarnedAt(transient, answeringAddress, time.Minute)
	stalest := sourceOver(presenceOver(movedPeer, transientPeer)).StalestPeers(
		t.Context(),
		[]peerdirectory.KnownPeer{
			knownPeer(moved, answeringAddress, anotherAddress),
			knownPeer(transient, answeringAddress),
		},
		1,
	)

	if len(stalest) != 1 || stalest[0] != transient {
		t.Fatalf("StalestPeers = %v, want %s", stalest, transient)
	}
}

func TestAPeerAdmittedTooRecentlyToBeProbedIsNeverTheStalest(t *testing.T) {
	t.Parallel()

	newcomer, transient := hashOf(t, 'a'), hashOf(t, 'b')
	transientPeer := presenceEarnedAt(transient, answeringAddress, time.Minute)
	justAdmitted := knownPeer(newcomer, answeringAddress)
	justAdmitted.AdmittedAt = theInstant
	stalest := sourceOver(presenceOver(transientPeer)).StalestPeers(
		t.Context(),
		[]peerdirectory.KnownPeer{justAdmitted, knownPeer(transient, answeringAddress)},
		1,
	)

	if len(stalest) != 1 || stalest[0] != transient {
		t.Fatalf("StalestPeers = %v, want %s", stalest, transient)
	}
}

func TestPeersThatEarnedNoPresenceAreStalestInTheOrderTheyAnswered(t *testing.T) {
	t.Parallel()

	recent, old := hashOf(t, 'a'), hashOf(t, 'b')
	answeredRecently, answeredLongAgo := knownPeer(recent, answeringAddress),
		knownPeer(old, answeringAddress)
	answeredRecently.AnsweredAt = theInstant.Add(-time.Minute)
	answeredLongAgo.AnsweredAt = theInstant.Add(-time.Hour)
	stalest := sourceOver(nil).StalestPeers(
		t.Context(),
		[]peerdirectory.KnownPeer{answeredRecently, answeredLongAgo},
		1,
	)

	if len(stalest) != 1 || stalest[0] != old {
		t.Fatalf("StalestPeers = %v, want %s", stalest, old)
	}
}

func TestNoPeerIsStalestWhenNoneIsAskedFor(t *testing.T) {
	t.Parallel()

	stalest := sourceOver(nil).StalestPeers(
		t.Context(),
		[]peerdirectory.KnownPeer{knownPeer(hashOf(t, 'a'), answeringAddress)},
		0,
	)

	if stalest != nil {
		t.Fatalf("StalestPeers = %v, want none", stalest)
	}
}
