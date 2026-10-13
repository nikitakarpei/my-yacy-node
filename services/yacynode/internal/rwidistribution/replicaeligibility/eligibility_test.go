package replicaeligibility_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicaeligibility"
)

type clock struct{ at time.Time }

func (c *clock) now() time.Time { return c.at }

func TestUnknownPeerHasNoCooldownLeft(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := replicaeligibility.New(time.Minute, clk.now)

	if left := peers.CooldownLeftOf(yacymodel.WordHash("peer")); left != 0 {
		t.Fatalf("CooldownLeftOf = %v, want 0 for a peer that was never offered a posting", left)
	}
}

func TestDeclinedPeerHasTheCooldownLeft(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := replicaeligibility.New(time.Minute, clk.now)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)

	if left := peers.CooldownLeftOf(peer); left != time.Minute {
		t.Fatalf("CooldownLeftOf = %v, want the whole cooldown %v", left, time.Minute)
	}
}

func TestRequestedPauseOutlastsCooldown(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := replicaeligibility.New(time.Minute, clk.now)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, time.Hour)

	if left := peers.CooldownLeftOf(peer); left != time.Hour {
		t.Fatalf("CooldownLeftOf = %v, want the requested pause %v", left, time.Hour)
	}
}

func TestCooldownLeftShrinksAsTimePasses(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := replicaeligibility.New(time.Hour, clk.now)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)
	clk.at = clk.at.Add(20 * time.Minute)

	if left := peers.CooldownLeftOf(peer); left != 40*time.Minute {
		t.Fatalf("CooldownLeftOf = %v, want %v", left, 40*time.Minute)
	}
}

func TestCooldownEndsAfterItsDuration(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := replicaeligibility.New(time.Minute, clk.now)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)
	clk.at = clk.at.Add(time.Minute)

	if left := peers.CooldownLeftOf(peer); left != 0 {
		t.Fatalf("CooldownLeftOf = %v, want 0 once the cooldown has passed", left)
	}
}

func TestAcceptanceEndsCooldown(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := replicaeligibility.New(time.Hour, clk.now)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)
	peers.OfferAccepted(peer)

	if left := peers.CooldownLeftOf(peer); left != 0 {
		t.Fatalf("CooldownLeftOf = %v, want 0 after the peer accepted an offer", left)
	}
}
