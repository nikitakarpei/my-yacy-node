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

func testPosting() yacymodel.RWIPosting {
	url, err := yacymodel.ParseURLHash(yacymodel.WordHash("u1").String())
	if err != nil {
		panic(err)
	}

	return yacymodel.RWIPosting{WordHash: yacymodel.WordHash("w1"), URLHash: url}
}

func closerPeer() yacymodel.Hash { return yacymodel.WordHash("peer") }

func otherCloserPeer() yacymodel.Hash { return yacymodel.WordHash("p1") }

func thisNodeFartherThanEveryPeer() yacymodel.Hash { return yacymodel.WordHash("self5") }

func thisNodeCloserThanEveryPeer() yacymodel.Hash { return yacymodel.WordHash("self22") }

func openHandoff(
	self yacymodel.Hash,
	redundancy int,
) (*postinghandoff.Handoff, *fakePostingPurger) {
	purger := &fakePostingPurger{}

	return postinghandoff.New(purger, postinghandoff.Config{
		Self:       self,
		Redundancy: redundancy,
	}), purger
}

func handOff(
	t *testing.T,
	handoff *postinghandoff.Handoff,
	holders []yacymodel.Hash,
	responsiblePeers []yacymodel.Hash,
) bool {
	t.Helper()

	wasHandedOff, err := handoff.HandOffIfHeldByResponsiblePeers(
		context.Background(), nil, testPosting(), testPostingPosition(t), holders, responsiblePeers,
	)
	if err != nil {
		t.Fatalf("HandOffIfHeldByResponsiblePeers: %v", err)
	}

	return wasHandedOff
}

func testPostingPosition(t *testing.T) yacymodel.DHTRingPosition {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}

	return yacymodel.DHTRingPositionOfPosting(testPosting(), partitions)
}

func TestPostingHandedOffOnceEveryResponsiblePeerHoldsIt(t *testing.T) {
	handoff, purger := openHandoff(thisNodeFartherThanEveryPeer(), handoffRedundancy)

	responsiblePeers := []yacymodel.Hash{closerPeer()}
	if !handOff(t, handoff, responsiblePeers, responsiblePeers) {
		t.Fatal("posting kept, want it handed off: redundancy is met by a responsible peer")
	}
	if len(purger.purgedPostings) != 1 {
		t.Fatalf("purged = %+v, want the handed-off posting", purger.purgedPostings)
	}
}

func TestPostingKeptBelowRedundancy(t *testing.T) {
	handoff, purger := openHandoff(thisNodeFartherThanEveryPeer(), 2)

	responsiblePeers := []yacymodel.Hash{closerPeer(), otherCloserPeer()}
	if handOff(t, handoff, []yacymodel.Hash{closerPeer()}, responsiblePeers) {
		t.Fatal("posting handed off, want it kept: only one of two owed peers holds it")
	}
	if len(purger.purgedPostings) != 0 {
		t.Fatalf("purged = %+v, want none", purger.purgedPostings)
	}
}

func TestPostingKeptWhileOnlyCloserPeersBeyondTheResponsiblePeersHoldIt(t *testing.T) {
	handoff, purger := openHandoff(thisNodeFartherThanEveryPeer(), handoffRedundancy)

	if handOff(
		t, handoff, []yacymodel.Hash{otherCloserPeer()}, []yacymodel.Hash{closerPeer()},
	) {
		t.Fatal("posting handed off, want it kept: the holder is not a responsible peer")
	}
	if len(purger.purgedPostings) != 0 {
		t.Fatalf("purged = %+v, want none", purger.purgedPostings)
	}
}

func TestPostingKeptWhileTheResponsibleHolderIsNotCloserThanThisNode(t *testing.T) {
	handoff, _ := openHandoff(thisNodeCloserThanEveryPeer(), handoffRedundancy)

	responsiblePeers := []yacymodel.Hash{closerPeer()}
	if handOff(t, handoff, responsiblePeers, responsiblePeers) {
		t.Fatal("posting handed off, want it kept: this node is closer than the holder")
	}
}

func TestPostingGoneFromTheIndexIsNotHandedOff(t *testing.T) {
	handoff, purger := openHandoff(thisNodeFartherThanEveryPeer(), handoffRedundancy)
	purger.postingGone = true

	responsiblePeers := []yacymodel.Hash{closerPeer()}
	if handOff(t, handoff, responsiblePeers, responsiblePeers) {
		t.Fatal("posting handed off, want nothing: the index no longer holds it")
	}
}

func TestKeepEveryPostingHandsOffNothing(t *testing.T) {
	responsiblePeers := []yacymodel.Hash{closerPeer()}
	wasHandedOff, err := postinghandoff.KeepEveryPosting{}.HandOffIfHeldByResponsiblePeers(
		context.Background(),
		nil,
		testPosting(),
		testPostingPosition(t),
		responsiblePeers,
		responsiblePeers,
	)
	if err != nil {
		t.Fatalf("HandOffIfHeldByResponsiblePeers: %v", err)
	}
	if wasHandedOff {
		t.Fatal("posting handed off, want every posting kept")
	}
}
