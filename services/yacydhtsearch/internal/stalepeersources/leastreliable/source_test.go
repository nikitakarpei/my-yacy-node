package leastreliable_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
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

type reliabilityPerPeerAtAddress map[probeanswerhistory.PeerAtAddress]float64

func (reliability reliabilityPerPeerAtAddress) ReliabilityOf(
	_ context.Context,
	peerAtAddress probeanswerhistory.PeerAtAddress,
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
		probeanswerhistory.PeerAtAddress{Hash: peer, Address: address}: reliability,
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

func candidatePeer(peer yacymodel.Hash, addresses ...string) peerdirectory.CandidatePeer {
	return peerdirectory.CandidatePeer{Hash: peer, Addresses: addresses}
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
	).StalestPeersFirst(
		t.Context(),
		[]peerdirectory.KnownPeer{
			knownPeer(present, answeringAddress),
			knownPeer(transient, answeringAddress),
		},
		nil,
	)

	if len(stalest) != 2 || stalest[0] != transient {
		t.Fatalf("StalestPeersFirst = %v, want %s first", stalest, transient)
	}
}

func TestAPeerSpeaksForItselfThroughTheAddressItHasDoneBestFrom(t *testing.T) {
	t.Parallel()

	moved, transient := hashOf(t, 'a'), hashOf(t, 'b')
	stalest := sourceOver(
		reliabilityEarnedAt(moved, anotherAddress, 1),
		reliabilityEarnedAt(transient, answeringAddress, 0.1),
	).StalestPeersFirst(
		t.Context(),
		[]peerdirectory.KnownPeer{
			knownPeer(moved, answeringAddress, anotherAddress),
			knownPeer(transient, answeringAddress),
		},
		nil,
	)

	if len(stalest) != 2 || stalest[0] != transient {
		t.Fatalf("StalestPeersFirst = %v, want %s first", stalest, transient)
	}
}

func TestAPeerAdmittedTooRecentlyToBeProbedIsNeverTheStalest(t *testing.T) {
	t.Parallel()

	newcomer, transient := hashOf(t, 'a'), hashOf(t, 'b')
	justAdmitted := knownPeer(newcomer, answeringAddress)
	justAdmitted.AdmittedAt = theInstant
	stalest := sourceOver(
		reliabilityEarnedAt(transient, answeringAddress, 0.1),
	).StalestPeersFirst(
		t.Context(),
		[]peerdirectory.KnownPeer{justAdmitted, knownPeer(transient, answeringAddress)},
		nil,
	)

	if len(stalest) != 2 || stalest[0] != transient {
		t.Fatalf("StalestPeersFirst = %v, want %s first", stalest, transient)
	}
}

func TestPeersThatEarnedNoReliabilityAreStalestInTheOrderTheyAnswered(t *testing.T) {
	t.Parallel()

	recent, old := hashOf(t, 'a'), hashOf(t, 'b')
	answeredRecently, answeredLongAgo := knownPeer(recent, answeringAddress),
		knownPeer(old, answeringAddress)
	answeredRecently.AnsweredAt = theInstant.Add(-time.Minute)
	answeredLongAgo.AnsweredAt = theInstant.Add(-time.Hour)
	stalest := sourceOver().StalestPeersFirst(
		t.Context(),
		[]peerdirectory.KnownPeer{answeredRecently, answeredLongAgo},
		nil,
	)

	if len(stalest) != 2 || stalest[0] != old {
		t.Fatalf("StalestPeersFirst = %v, want %s first", stalest, old)
	}
}

func TestAnOfferedPeerIsStalerThanAHeldPeerOfTheSameReliability(t *testing.T) {
	t.Parallel()

	held, offered := hashOf(t, 'a'), hashOf(t, 'b')
	stalest := sourceOver(
		reliabilityEarnedAt(held, answeringAddress, 0.5),
		reliabilityEarnedAt(offered, answeringAddress, 0.5),
	).StalestPeersFirst(
		t.Context(),
		[]peerdirectory.KnownPeer{knownPeer(held, answeringAddress)},
		[]peerdirectory.CandidatePeer{candidatePeer(offered, answeringAddress)},
	)

	if len(stalest) != 2 || stalest[0] != offered {
		t.Fatalf("StalestPeersFirst = %v, want the offered %s first", stalest, offered)
	}
}

func TestAnOfferedPeerThisDeploymentHasSeenAnswerOutranksAHeldPeerItHasNot(t *testing.T) {
	t.Parallel()

	held, offered := hashOf(t, 'a'), hashOf(t, 'b')
	stalest := sourceOver(
		reliabilityEarnedAt(offered, answeringAddress, 0.5),
	).StalestPeersFirst(
		t.Context(),
		[]peerdirectory.KnownPeer{knownPeer(held, answeringAddress)},
		[]peerdirectory.CandidatePeer{candidatePeer(offered, answeringAddress)},
	)

	if len(stalest) != 2 || stalest[0] != held {
		t.Fatalf("StalestPeersFirst = %v, want the held %s first", stalest, held)
	}
}

func TestNoPeerIsStalestWhenTheDirectoryIsEmptyAndNothingIsOffered(t *testing.T) {
	t.Parallel()

	if stalest := sourceOver().StalestPeersFirst(t.Context(), nil, nil); len(stalest) != 0 {
		t.Fatalf("StalestPeersFirst = %v, want none", stalest)
	}
}
