package peereligibility_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/peereligibility"
)

type clock struct{ at time.Time }

func (c *clock) now() time.Time { return c.at }

type fakeObserver struct {
	amountOfExclusions int
	sendTimes          []time.Duration
	peersAtSendLimit   int
}

func (f *fakeObserver) ObservePeerExcluded() { f.amountOfExclusions++ }

func (f *fakeObserver) ObservePeerSendTime(sendTime time.Duration) {
	f.sendTimes = append(f.sendTimes, sendTime)
}

func (f *fakeObserver) ObservePeersAtSendLimit(amountOfPeers int) {
	f.peersAtSendLimit = amountOfPeers
}

func peersWith(cooldown time.Duration, clk *clock) *peereligibility.Peers {
	return peereligibility.New(
		peereligibility.Config{
			Cooldown:                    cooldown,
			FailedOffersBeforeExclusion: 3,
			ExclusionDuration:           time.Hour,
		},
		clk.now,
		&fakeObserver{},
	)
}

func assertCooldownLasts(
	t *testing.T,
	peers *peereligibility.Peers,
	clk *clock,
	peer yacymodel.Hash,
	cooldown time.Duration,
) {
	t.Helper()

	cooldownStart := clk.at
	clk.at = cooldownStart.Add(cooldown - time.Second)
	if !peers.IsInCooldown(peer) {
		t.Fatalf("IsInCooldown = false, want true before %v pass", cooldown)
	}
	clk.at = cooldownStart.Add(cooldown)
	if peers.IsInCooldown(peer) {
		t.Fatalf("IsInCooldown = true, want false once %v pass", cooldown)
	}
}

func TestUnknownPeerIsNotInCooldown(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)

	if peers.IsInCooldown(yacymodel.WordHash("peer")) {
		t.Fatal("IsInCooldown = true, want false for a peer that was never offered a posting")
	}
}

func TestDeclinedPeerIsInCooldownForTheCooldown(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)

	assertCooldownLasts(t, peers, clk, peer, time.Minute)
}

func TestRequestedPauseOutlastsCooldown(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, time.Hour)

	assertCooldownLasts(t, peers, clk, peer, time.Hour)
}

func TestAcceptanceEndsCooldown(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Hour, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)
	peers.OfferAccepted(peer)

	if peers.IsInCooldown(peer) {
		t.Fatal("IsInCooldown = true, want false after the peer accepted an offer")
	}
}

type exclusionHarness struct {
	clk      *clock
	counter  *fakeObserver
	peers    *peereligibility.Peers
	peer     yacymodel.Hash
	cooldown time.Duration
}

func newExclusionHarness() *exclusionHarness {
	h := &exclusionHarness{
		clk:      &clock{at: time.Unix(1000, 0)},
		counter:  &fakeObserver{},
		peer:     yacymodel.WordHash("peer"),
		cooldown: time.Minute,
	}
	h.peers = peereligibility.New(
		peereligibility.Config{
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
	if h.counter.amountOfExclusions != 0 {
		t.Fatalf("exclusions = %d, want 0", h.counter.amountOfExclusions)
	}
	assertCooldownLasts(t, h.peers, h.clk, h.peer, h.cooldown)
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
