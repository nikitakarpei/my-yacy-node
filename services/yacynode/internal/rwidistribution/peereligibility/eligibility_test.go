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

func peersWith(leastPause time.Duration, clk *clock) *peereligibility.Peers {
	return peereligibility.New(
		peereligibility.Config{
			LeastPause:                  leastPause,
			FailedOffersBeforeExclusion: 3,
			ExclusionDuration:           time.Hour,
		},
		clk.now,
		&fakeObserver{},
	)
}

func assertPauseLasts(
	t *testing.T,
	peers *peereligibility.Peers,
	clk *clock,
	peer yacymodel.Hash,
	leastPause time.Duration,
) {
	t.Helper()

	pauseStart := clk.at
	clk.at = pauseStart.Add(leastPause - time.Second)
	if !peers.IsPaused(peer) {
		t.Fatalf("IsPaused = false, want true before %v pass", leastPause)
	}
	clk.at = pauseStart.Add(leastPause)
	if peers.IsPaused(peer) {
		t.Fatalf("IsPaused = true, want false once %v pass", leastPause)
	}
}

func TestUnknownPeerIsNotPaused(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)

	if peers.IsPaused(yacymodel.WordHash("peer")) {
		t.Fatal("IsPaused = true, want false for a peer that was never offered a posting")
	}
}

func TestDeclinedPeerIsPausedForTheLeastPause(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)

	assertPauseLasts(t, peers, clk, peer, time.Minute)
}

func TestRequestedPauseOutlastsLeastPause(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, time.Hour)

	assertPauseLasts(t, peers, clk, peer, time.Hour)
}

func TestAcceptanceEndsPause(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Hour, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(peer, 0)
	peers.OfferAccepted(peer)

	if peers.IsPaused(peer) {
		t.Fatal("IsPaused = true, want false after the peer accepted an offer")
	}
}

type exclusionHarness struct {
	clk        *clock
	counter    *fakeObserver
	peers      *peereligibility.Peers
	peer       yacymodel.Hash
	leastPause time.Duration
}

func newExclusionHarness() *exclusionHarness {
	h := &exclusionHarness{
		clk:        &clock{at: time.Unix(1000, 0)},
		counter:    &fakeObserver{},
		peer:       yacymodel.WordHash("peer"),
		leastPause: time.Minute,
	}
	h.peers = peereligibility.New(
		peereligibility.Config{
			LeastPause:                  h.leastPause,
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
	assertPauseLasts(t, h.peers, h.clk, h.peer, h.leastPause)
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
