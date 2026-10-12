package replicaeligibility_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicaeligibility"
)

type clock struct{ at time.Time }

func (c *clock) now() time.Time { return c.at }

type exclusionCounter struct{ amountOfExclusions int }

func (c *exclusionCounter) ObservePeerExcluded() { c.amountOfExclusions++ }

func peersWith(cooldown time.Duration, clk *clock) *replicaeligibility.Peers {
	return replicaeligibility.New(
		replicaeligibility.Config{
			Cooldown:                    cooldown,
			FailedOffersBeforeExclusion: 3,
			ExclusionDuration:           time.Hour,
		},
		clk.now,
		&exclusionCounter{},
	)
}

func TestUnknownPeerHasNoCooldownLeft(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)

	if left := peers.CooldownLeftOf(yacymodel.WordHash("peer")); left != 0 {
		t.Fatalf("CooldownLeftOf = %v, want 0 for a peer that was never offered a posting", left)
	}
}

func TestDeclinedPeerHasTheCooldownLeft(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)

	if left := peers.CooldownLeftOf(peer); left != time.Minute {
		t.Fatalf("CooldownLeftOf = %v, want the whole cooldown %v", left, time.Minute)
	}
}

func TestRequestedPauseOutlastsCooldown(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, time.Hour)

	if left := peers.CooldownLeftOf(peer); left != time.Hour {
		t.Fatalf("CooldownLeftOf = %v, want the requested pause %v", left, time.Hour)
	}
}

func TestCooldownLeftShrinksAsTimePasses(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Hour, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)
	clk.at = clk.at.Add(20 * time.Minute)

	if left := peers.CooldownLeftOf(peer); left != 40*time.Minute {
		t.Fatalf("CooldownLeftOf = %v, want %v", left, 40*time.Minute)
	}
}

func TestCooldownEndsAfterItsDuration(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)
	clk.at = clk.at.Add(time.Minute)

	if left := peers.CooldownLeftOf(peer); left != 0 {
		t.Fatalf("CooldownLeftOf = %v, want 0 once the cooldown has passed", left)
	}
}

func TestAcceptanceEndsCooldown(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Hour, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)
	peers.OfferAccepted(peer)

	if left := peers.CooldownLeftOf(peer); left != 0 {
		t.Fatalf("CooldownLeftOf = %v, want 0 after the peer accepted an offer", left)
	}
}

type exclusionHarness struct {
	clk      *clock
	counter  *exclusionCounter
	peers    *replicaeligibility.Peers
	peer     yacymodel.Hash
	cooldown time.Duration
}

func newExclusionHarness() *exclusionHarness {
	h := &exclusionHarness{
		clk:      &clock{at: time.Unix(1000, 0)},
		counter:  &exclusionCounter{},
		peer:     yacymodel.WordHash("peer"),
		cooldown: time.Minute,
	}
	h.peers = replicaeligibility.New(
		replicaeligibility.Config{
			Cooldown:                    h.cooldown,
			FailedOffersBeforeExclusion: 3,
			ExclusionDuration:           time.Hour,
		},
		h.clk.now,
		h.counter,
	)

	return h
}

func (h *exclusionHarness) failOffers(amountOfFailedOffers int) {
	for range amountOfFailedOffers {
		h.peers.OfferFailed(context.Background(), h.peer)
	}
}

func TestFailedOffersBelowTheThresholdOnlyCoolThePeerDown(t *testing.T) {
	h := newExclusionHarness()

	h.failOffers(2)

	if h.peers.IsExcluded(h.peer) {
		t.Fatal("IsExcluded = true, want false below the threshold")
	}
	if left := h.peers.CooldownLeftOf(h.peer); left != h.cooldown {
		t.Fatalf("CooldownLeftOf = %v, want the cooldown %v", left, h.cooldown)
	}
	if h.counter.amountOfExclusions != 0 {
		t.Fatalf("exclusions = %d, want 0", h.counter.amountOfExclusions)
	}
}

func TestFailedOffersInARowExcludeThePeerForTheExclusionDuration(t *testing.T) {
	h := newExclusionHarness()

	h.failOffers(3)

	if !h.peers.IsExcluded(h.peer) {
		t.Fatal("IsExcluded = false, want true at the threshold")
	}
	if h.counter.amountOfExclusions != 1 {
		t.Fatalf("exclusions = %d, want 1", h.counter.amountOfExclusions)
	}
	h.clk.at = h.clk.at.Add(time.Hour - time.Second)
	if !h.peers.IsExcluded(h.peer) {
		t.Fatal("IsExcluded = false, want true before the exclusion ends")
	}
	h.clk.at = h.clk.at.Add(time.Second)
	if h.peers.IsExcluded(h.peer) {
		t.Fatal("IsExcluded = true, want false once the exclusion ends")
	}
}

func TestFailedOffersDuringAnExclusionDoNotExcludeAgain(t *testing.T) {
	h := newExclusionHarness()

	h.failOffers(5)

	if h.counter.amountOfExclusions != 1 {
		t.Fatalf("exclusions = %d, want 1", h.counter.amountOfExclusions)
	}
}

func TestAcceptanceEndsExclusionAndStartsTheFailedOffersOver(t *testing.T) {
	h := newExclusionHarness()
	h.failOffers(3)

	h.peers.OfferAccepted(h.peer)

	if h.peers.IsExcluded(h.peer) {
		t.Fatal("IsExcluded = true, want false after the peer accepted an offer")
	}
	h.failOffers(2)
	if h.peers.IsExcluded(h.peer) {
		t.Fatal("IsExcluded = true, want false: acceptance starts the count over")
	}
}

func TestDeclineNeitherCountsNorStartsTheFailedOffersOver(t *testing.T) {
	h := newExclusionHarness()
	h.failOffers(2)

	h.peers.OfferDeclined(h.peer, 0)

	if h.peers.IsExcluded(h.peer) {
		t.Fatal("IsExcluded = true, want false: a decline is not a failed offer")
	}
	h.failOffers(1)
	if !h.peers.IsExcluded(h.peer) {
		t.Fatal("IsExcluded = false, want true: a decline keeps the count")
	}
}

func TestAFailedOfferAfterAnEndedExclusionExcludesAgain(t *testing.T) {
	h := newExclusionHarness()
	h.failOffers(3)
	h.clk.at = h.clk.at.Add(time.Hour)

	h.failOffers(1)

	if !h.peers.IsExcluded(h.peer) {
		t.Fatal("IsExcluded = false, want true after one more failed offer")
	}
	if h.counter.amountOfExclusions != 2 {
		t.Fatalf("exclusions = %d, want 2", h.counter.amountOfExclusions)
	}
}
