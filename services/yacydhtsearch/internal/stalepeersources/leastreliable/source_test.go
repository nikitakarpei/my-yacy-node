package leastreliable_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
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

type reliabilityPerPeerAtAddress map[peeranswerhistory.PeerAtAddress]float64

func (reliability reliabilityPerPeerAtAddress) ReliabilityOf(
	_ context.Context,
	peerAtAddress peeranswerhistory.PeerAtAddress,
) float64 {
	return reliability[peerAtAddress]
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

func reliabilityEarnedAt(
	peer yacymodel.Hash,
	address string,
	reliability float64,
) reliabilityPerPeerAtAddress {
	return reliabilityPerPeerAtAddress{
		peeranswerhistory.PeerAtAddress{Hash: peer, Address: address}: reliability,
	}
}

func sourceOver(earned ...reliabilityPerPeerAtAddress) leastreliable.Source {
	reliability := reliabilityPerPeerAtAddress{}
	for _, earnedByOnePeer := range earned {
		for peerAtAddress, earnedReliability := range earnedByOnePeer {
			reliability[peerAtAddress] = earnedReliability
		}
	}

	return leastreliable.New(
		reliability,
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
	stalest := sourceOver(
		reliabilityEarnedAt(present, answeringAddress, 1),
		reliabilityEarnedAt(transient, answeringAddress, 0.1),
	).StalestPeers(
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
	stalest := sourceOver(
		reliabilityEarnedAt(moved, anotherAddress, 1),
		reliabilityEarnedAt(transient, answeringAddress, 0.1),
	).StalestPeers(
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
	justAdmitted := knownPeer(newcomer, answeringAddress)
	justAdmitted.AdmittedAt = theInstant
	stalest := sourceOver(
		reliabilityEarnedAt(transient, answeringAddress, 0.1),
	).StalestPeers(
		t.Context(),
		[]peerdirectory.KnownPeer{justAdmitted, knownPeer(transient, answeringAddress)},
		1,
	)

	if len(stalest) != 1 || stalest[0] != transient {
		t.Fatalf("StalestPeers = %v, want %s", stalest, transient)
	}
}

func TestPeersThatEarnedNoReliabilityAreStalestInTheOrderTheyAnswered(t *testing.T) {
	t.Parallel()

	recent, old := hashOf(t, 'a'), hashOf(t, 'b')
	answeredRecently, answeredLongAgo := knownPeer(recent, answeringAddress),
		knownPeer(old, answeringAddress)
	answeredRecently.AnsweredAt = theInstant.Add(-time.Minute)
	answeredLongAgo.AnsweredAt = theInstant.Add(-time.Hour)
	stalest := sourceOver().StalestPeers(
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

	stalest := sourceOver().StalestPeers(
		t.Context(),
		[]peerdirectory.KnownPeer{knownPeer(hashOf(t, 'a'), answeringAddress)},
		0,
	)

	if stalest != nil {
		t.Fatalf("StalestPeers = %v, want none", stalest)
	}
}
