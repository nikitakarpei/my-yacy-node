package replicaeligibility

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type clock struct{ at time.Time }

func (c *clock) now() time.Time { return c.at }

func eligible(peers *Peers, peer yacymodel.Hash) bool {
	return len(peers.EligiblePeers([]yacymodel.Seed{{Hash: peer}})) == 1
}

func TestUnknownPeerIsEligible(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := New(time.Minute, clk.now)

	if !eligible(peers, yacymodel.WordHash("peer")) {
		t.Fatal("EligiblePeers = false, want true for a peer that was never offered a posting")
	}
}

func TestRefusedPeerIsHeldBackForCooldown(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := New(time.Minute, clk.now)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)

	if eligible(peers, peer) {
		t.Fatal("EligiblePeers = true, want false within the cooldown")
	}

	clk.at = clk.at.Add(time.Minute)

	if !eligible(peers, peer) {
		t.Fatal("EligiblePeers = false, want true once the cooldown has passed")
	}
	if heldBack := len(peers.heldBackUntil); heldBack != 0 {
		t.Fatalf("held back peers = %d, want the expired hold forgotten", heldBack)
	}
}

func TestRequestedPauseOutlastsCooldown(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := New(time.Minute, clk.now)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, time.Hour)

	clk.at = clk.at.Add(30 * time.Minute)

	if eligible(peers, peer) {
		t.Fatal("EligiblePeers = true, want false while the peer's own pause is unexpired")
	}
}

func TestAcceptanceClearsHoldBack(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := New(time.Hour, clk.now)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)
	peers.OfferAccepted(peer)

	if !eligible(peers, peer) {
		t.Fatal("EligiblePeers = false, want true after the peer accepted an offer")
	}
}
