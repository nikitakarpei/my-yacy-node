package postinghandoff_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postinghandoff"
)

const handoffRedundancy = 1

type fakePostingPurger struct {
	postingGone    bool
	purgedPostings []yacymodel.RWIPosting
}

func (f *fakePostingPurger) PurgePosting(
	_ *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (bool, error) {
	if f.postingGone {
		return false, nil
	}
	f.purgedPostings = append(f.purgedPostings, yacymodel.RWIPosting{WordHash: word, URLHash: url})

	return true, nil
}

type fakeReachability struct {
	reachablePeers []yacymodel.Hash
}

func (f fakeReachability) IsReachable(_ context.Context, peer yacymodel.Hash) bool {
	for _, reachable := range f.reachablePeers {
		if reachable == peer {
			return true
		}
	}

	return false
}

func testPosting() yacymodel.RWIPosting {
	url, err := yacymodel.ParseURLHash(yacymodel.WordHash("u1").String())
	if err != nil {
		panic(err)
	}

	return yacymodel.RWIPosting{WordHash: yacymodel.WordHash("w1"), URLHash: url}
}

func closerPeer() yacymodel.Hash { return yacymodel.WordHash("peer") }

func thisNodeFartherThanEveryPeer() yacymodel.Hash { return yacymodel.WordHash("self5") }

func openHandoff(
	t *testing.T,
	reachability postinghandoff.Reachability,
	redundancy int,
) (*postinghandoff.Handoff, *fakePostingPurger) {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}
	purger := &fakePostingPurger{}

	return postinghandoff.New(purger, reachability, postinghandoff.Config{
		Partitions: partitions,
		Self:       thisNodeFartherThanEveryPeer(),
		Redundancy: redundancy,
	}), purger
}

func handOff(
	t *testing.T,
	handoff *postinghandoff.Handoff,
	holders ...yacymodel.Hash,
) bool {
	t.Helper()

	wasHandedOff, err := handoff.HandOff(context.Background(), nil, testPosting(), holders)
	if err != nil {
		t.Fatalf("HandOff: %v", err)
	}

	return wasHandedOff
}

func TestPostingHandedOffOnceEnoughReachableCloserHoldersExist(t *testing.T) {
	handoff, purger := openHandoff(
		t, fakeReachability{reachablePeers: []yacymodel.Hash{closerPeer()}}, handoffRedundancy,
	)

	if !handOff(t, handoff, closerPeer()) {
		t.Fatal("posting kept, want it handed off: redundancy is met by a closer peer")
	}
	if len(purger.purgedPostings) != 1 {
		t.Fatalf("purged = %+v, want the handed-off posting", purger.purgedPostings)
	}
}

func TestPostingKeptBelowRedundancy(t *testing.T) {
	handoff, purger := openHandoff(
		t, fakeReachability{reachablePeers: []yacymodel.Hash{closerPeer()}}, 2,
	)

	if handOff(t, handoff, closerPeer()) {
		t.Fatal("posting handed off, want it kept: only one of two owed peers holds it")
	}
	if len(purger.purgedPostings) != 0 {
		t.Fatalf("purged = %+v, want none", purger.purgedPostings)
	}
}

func TestPostingKeptWhileTheCloserHolderIsUnreachable(t *testing.T) {
	handoff, _ := openHandoff(t, fakeReachability{}, handoffRedundancy)

	if handOff(t, handoff, closerPeer()) {
		t.Fatal("posting handed off, want it kept: the holder is unreachable")
	}
}

func TestPostingGoneFromTheIndexIsNotHandedOff(t *testing.T) {
	handoff, purger := openHandoff(
		t, fakeReachability{reachablePeers: []yacymodel.Hash{closerPeer()}}, handoffRedundancy,
	)
	purger.postingGone = true

	if handOff(t, handoff, closerPeer()) {
		t.Fatal("posting handed off, want nothing: the index no longer holds it")
	}
}

func TestKeepEveryPostingHandsOffNothing(t *testing.T) {
	wasHandedOff, err := postinghandoff.KeepEveryPosting{}.HandOff(
		context.Background(), nil, testPosting(), []yacymodel.Hash{closerPeer()},
	)
	if err != nil {
		t.Fatalf("HandOff: %v", err)
	}
	if wasHandedOff {
		t.Fatal("posting handed off, want every posting kept")
	}
}
