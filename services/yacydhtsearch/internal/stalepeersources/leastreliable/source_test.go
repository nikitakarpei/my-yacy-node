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
)

var theInstant = time.Unix(1_000_000, 0)

type observedPeers []presenceaccrual.ObservedPeer

func (peers observedPeers) ObservedPeers(context.Context) []presenceaccrual.ObservedPeer {
	return peers
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

func observedPeerOf(peer yacymodel.Hash, presence time.Duration) presenceaccrual.ObservedPeer {
	return presenceaccrual.ObservedPeer{
		PeerAtAddress:    presenceaccrual.PeerAtAddress{Hash: peer, Address: "10.0.0.1"},
		FirstAnsweredAt:  theInstant.Add(-presence),
		LatestAnsweredAt: theInstant,
		Presence:         presence,
	}
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

func TestThePeerWithTheLeastEarnedReliabilityIsTheStalest(t *testing.T) {
	t.Parallel()

	present, transient := hashOf(t, 'a'), hashOf(t, 'b')
	stalest := sourceOver(observedPeers{
		observedPeerOf(present, time.Hour),
		observedPeerOf(transient, time.Minute),
	}).StalestPeers(t.Context(), []peerdirectory.KnownPeer{
		{Hash: present, AdmittedAt: theInstant.Add(-longEnoughToProbe)},
		{Hash: transient, AdmittedAt: theInstant.Add(-longEnoughToProbe)},
	}, 1)

	if len(stalest) != 1 || stalest[0] != transient {
		t.Fatalf("StalestPeers = %v, want %s", stalest, transient)
	}
}

func TestAPeerAdmittedTooRecentlyToBeProbedIsNeverTheStalest(t *testing.T) {
	t.Parallel()

	newcomer, transient := hashOf(t, 'a'), hashOf(t, 'b')
	stalest := sourceOver(observedPeers{
		observedPeerOf(transient, time.Minute),
	}).StalestPeers(t.Context(), []peerdirectory.KnownPeer{
		{Hash: newcomer, AdmittedAt: theInstant},
		{Hash: transient, AdmittedAt: theInstant.Add(-longEnoughToProbe)},
	}, 1)

	if len(stalest) != 1 || stalest[0] != transient {
		t.Fatalf("StalestPeers = %v, want %s", stalest, transient)
	}
}

func TestPeersThatEarnedNoPresenceAreStalestInTheOrderTheyAnswered(t *testing.T) {
	t.Parallel()

	recent, old := hashOf(t, 'a'), hashOf(t, 'b')
	stalest := sourceOver(nil).StalestPeers(t.Context(), []peerdirectory.KnownPeer{
		{
			Hash:       recent,
			AnsweredAt: theInstant.Add(-time.Minute),
			AdmittedAt: theInstant.Add(-longEnoughToProbe),
		},
		{
			Hash:       old,
			AnsweredAt: theInstant.Add(-time.Hour),
			AdmittedAt: theInstant.Add(-longEnoughToProbe),
		},
	}, 1)

	if len(stalest) != 1 || stalest[0] != old {
		t.Fatalf("StalestPeers = %v, want %s", stalest, old)
	}
}

func TestNoPeerIsStalestWhenNoneIsAskedFor(t *testing.T) {
	t.Parallel()

	peer := hashOf(t, 'a')
	stalest := sourceOver(nil).StalestPeers(t.Context(), []peerdirectory.KnownPeer{
		{Hash: peer, AdmittedAt: theInstant.Add(-longEnoughToProbe)},
	}, 0)

	if stalest != nil {
		t.Fatalf("StalestPeers = %v, want none", stalest)
	}
}
