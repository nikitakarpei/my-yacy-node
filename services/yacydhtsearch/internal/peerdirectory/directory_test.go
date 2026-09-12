package peerdirectory_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	cooldown     = 5 * time.Second
	wideCapacity = 16
)

type silentObserver struct{}

func (silentObserver) PeerAdmitted(context.Context, yacymodel.Hash, int)     {}
func (silentObserver) PeerAnswering(context.Context, yacymodel.Hash, string) {}
func (silentObserver) PeerSilent(context.Context, yacymodel.Hash)            {}
func (silentObserver) PeerDropped(context.Context, yacymodel.Hash)           {}
func (silentObserver) DirectoryHolds(context.Context, int, int, int)         {}

type oldestAdmittedFirst struct{}

func (oldestAdmittedFirst) StalestPeers(
	known []peerdirectory.KnownPeer,
	limit int,
) []yacymodel.Hash {
	stalest := make([]yacymodel.Hash, 0, limit)
	oldest := known[0]
	for _, peer := range known {
		if peer.AdmittedAt.Before(oldest.AdmittedAt) {
			oldest = peer
		}
	}

	return append(stalest, oldest.Hash)
}

type testClock struct{ instant time.Time }

func (c *testClock) now() time.Time { return c.instant }

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

func seedOf(t *testing.T, hash yacymodel.Hash, host string) yacymodel.Seed {
	t.Helper()

	address, err := yacymodel.ParseHost(host)
	if err != nil {
		t.Fatalf("parse host %q: %v", host, err)
	}
	port, err := yacymodel.ParsePort("8090")
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	return yacymodel.Seed{
		Hash:           hash,
		PrimaryAddress: yacymodel.Some(address),
		Port:           yacymodel.Some(port),
	}
}

func directoryAt(clock *testClock, capacity int) *peerdirectory.Directory {
	return peerdirectory.New(
		capacity,
		cooldown,
		clock.now,
		oldestAdmittedFirst{},
		peerdirectory.DirectoryObservers{silentObserver{}},
	)
}

func TestAnAdmittedPeerIsNotAskedBeforeAnAddressAnswered(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, hashOf(t, 'a'), "10.0.0.1")})

	if askable := directory.AskablePeers(t.Context()); len(askable) != 0 {
		t.Fatalf("AskablePeers = %v, want none before a probe answered", askable)
	}
}

func TestAPeerBecomesAskableOnTheAddressThatAnswered(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	peer := hashOf(t, 'a')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.1")})
	directory.ConfirmAnswering(t.Context(), peer, "http://10.0.0.1:8090")

	askable := directory.AskablePeers(t.Context())
	if len(askable) != 1 || askable[0].Address != "http://10.0.0.1:8090" {
		t.Fatalf("AskablePeers = %v, want the address that answered", askable)
	}
}

func TestAnAskedPeerRestsForTheCooldown(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	peer := hashOf(t, 'a')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.1")})
	directory.ConfirmAnswering(t.Context(), peer, "http://10.0.0.1:8090")

	directory.MarkPeersChosen(t.Context(), directory.AskablePeers(t.Context()))
	clock.instant = clock.instant.Add(cooldown - time.Second)
	if askable := directory.AskablePeers(t.Context()); len(askable) != 0 {
		t.Fatalf("AskablePeers = %v, want none inside the cooldown", askable)
	}

	clock.instant = clock.instant.Add(2 * time.Second)
	if askable := directory.AskablePeers(t.Context()); len(askable) != 1 {
		t.Fatalf("AskablePeers = %v, want the peer back after the cooldown", askable)
	}
}

func TestASilentPeerLeavesTheAskableSet(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	peer := hashOf(t, 'a')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.1")})
	directory.ConfirmAnswering(t.Context(), peer, "http://10.0.0.1:8090")
	directory.ConfirmSilent(t.Context(), peer)

	if askable := directory.AskablePeers(t.Context()); len(askable) != 0 {
		t.Fatalf("AskablePeers = %v, want none once the peer went silent", askable)
	}
}

func TestAdmittingAPeerAgainReplacesTheAddressesItAdvertises(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	peer := hashOf(t, 'a')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.1")})
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.2")})

	known := directory.KnownPeers(t.Context())
	if len(known) != 1 || known[0].Addresses[0] != "http://10.0.0.2:8090" {
		t.Fatalf("KnownPeers = %+v, want one peer on 10.0.0.2", known)
	}
}

func TestASeedWithoutAPortIsNotAPeerTheDirectoryKnows(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	directory.Admit(t.Context(), []yacymodel.Seed{{Hash: hashOf(t, 'a')}})

	if known := directory.KnownPeers(t.Context()); len(known) != 0 {
		t.Fatalf("KnownPeers = %+v, want none", known)
	}
}

func TestAFullDirectoryDropsTheStalestPeerToAdmitANewOne(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, 1)
	first, second := hashOf(t, 'a'), hashOf(t, 'b')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, first, "10.0.0.1")})
	clock.instant = clock.instant.Add(time.Minute)
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, second, "10.0.0.2")})

	known := directory.KnownPeers(t.Context())
	if len(known) != 1 || known[0].Hash != second {
		t.Fatalf("KnownPeers = %+v, want only the newly admitted peer", known)
	}
}

type contentsRecorder struct {
	silentObserver
	peers          int
	answeringPeers int
}

func (r *contentsRecorder) DirectoryHolds(_ context.Context, peers, answeringPeers, _ int) {
	r.peers = peers
	r.answeringPeers = answeringPeers
}

func TestTheDirectoryReportsHowManyOfItsPeersAnswer(t *testing.T) {
	t.Parallel()

	recorder := &contentsRecorder{}
	directory := peerdirectory.New(
		wideCapacity,
		cooldown,
		(&testClock{instant: time.Unix(0, 0)}).now,
		oldestAdmittedFirst{},
		peerdirectory.DirectoryObservers{recorder},
	)
	answering, silent := hashOf(t, 'a'), hashOf(t, 'b')
	directory.Admit(t.Context(), []yacymodel.Seed{
		seedOf(t, answering, "10.0.0.1"),
		seedOf(t, silent, "10.0.0.2"),
	})
	directory.ConfirmAnswering(t.Context(), answering, "http://10.0.0.1:8090")
	directory.ConfirmSilent(t.Context(), silent)

	if recorder.peers != 2 || recorder.answeringPeers != 1 {
		t.Fatalf("DirectoryHolds = %d peers, %d answering, want 2 and 1",
			recorder.peers, recorder.answeringPeers)
	}
}
