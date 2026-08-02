package replicarecipients

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
)

type clock struct{ at time.Time }

func (c *clock) now() time.Time { return c.at }

func TestUnknownPeerIsEligible(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	recipients := New(time.Minute, clk.now)

	if !recipients.Eligible(yacymodel.WordHash("peer")) {
		t.Fatal("Eligible = false, want true for a peer that was never offered a posting")
	}
}

func TestRefusedPeerIsHeldBackForCooldown(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	recipients := New(time.Minute, clk.now)
	peer := yacymodel.WordHash("peer")

	recipients.OfferAnswered(peer, postingcourier.Refused, 0)

	if recipients.Eligible(peer) {
		t.Fatal("Eligible = true, want false within the cooldown")
	}
	if held := recipients.IneligiblePeers(); held != 1 {
		t.Fatalf("IneligiblePeers = %d, want 1", held)
	}

	clk.at = clk.at.Add(time.Minute)

	if !recipients.Eligible(peer) {
		t.Fatal("Eligible = false, want true once the cooldown has passed")
	}
	if held := recipients.IneligiblePeers(); held != 0 {
		t.Fatalf("IneligiblePeers = %d, want 0 once the cooldown has passed", held)
	}
}

func TestRequestedPauseOutlastsCooldown(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	recipients := New(time.Minute, clk.now)
	peer := yacymodel.WordHash("peer")

	recipients.OfferAnswered(peer, postingcourier.Deferred, time.Hour)

	clk.at = clk.at.Add(30 * time.Minute)

	if recipients.Eligible(peer) {
		t.Fatal("Eligible = true, want false while the peer's own pause is unexpired")
	}
}

func TestAcceptanceClearsHoldBack(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	recipients := New(time.Hour, clk.now)
	peer := yacymodel.WordHash("peer")

	recipients.OfferAnswered(peer, postingcourier.Unreachable, 0)
	recipients.OfferAnswered(peer, postingcourier.Accepted, 0)

	if !recipients.Eligible(peer) {
		t.Fatal("Eligible = false, want true after the peer accepted an offer")
	}
}
