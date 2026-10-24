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
	exclusionCauses    []string
	sendTimes          []time.Duration
	peersAtSendLimit   int
}

func (f *fakeObserver) ObservePeerExcluded(cause string) {
	f.amountOfExclusions++
	f.exclusionCauses = append(f.exclusionCauses, cause)
}

func (f *fakeObserver) ObservePeerSendTime(sendTime time.Duration) {
	f.sendTimes = append(f.sendTimes, sendTime)
}

func (f *fakeObserver) ObservePeersAtSendLimit(amountOfPeers int) {
	f.peersAtSendLimit = amountOfPeers
}

func peersWith(leastPause time.Duration, clk *clock) *peereligibility.Peers {
	return peereligibility.New(
		peereligibility.Config{
			LeastPause:                      leastPause,
			UnacceptedOffersBeforeExclusion: 3,
			ExclusionDuration:               time.Hour,
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

	peers.OfferDeclined(context.Background(), peer, 0)

	assertPauseLasts(t, peers, clk, peer, time.Minute)
}

func TestRequestedPauseOutlastsLeastPause(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(context.Background(), peer, time.Hour)

	assertPauseLasts(t, peers, clk, peer, time.Hour)
}

func TestAcceptanceEndsPause(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Hour, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(context.Background(), peer, 0)
	peers.OfferAccepted(peer)

	if peers.IsPaused(peer) {
		t.Fatal("IsPaused = true, want false after the peer accepted an offer")
	}
}

func TestDeclinedOfferGivesADeclinedPauseEndingAtTheRequestedPause(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(context.Background(), peer, time.Hour)

	pause, paused := peers.PauseOf(peer)
	want := peereligibility.Pause{
		EndsAt: clk.at.Add(time.Hour),
		Cause:  peereligibility.PauseDeclined,
	}
	if !paused || pause != want {
		t.Fatalf("PauseOf = %+v, %v, want %+v", pause, paused, want)
	}
}

func TestDeclinedOfferPausesAtLeastTheLeastPause(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferDeclined(context.Background(), peer, time.Second)

	pause, paused := peers.PauseOf(peer)
	if !paused || !pause.EndsAt.Equal(clk.at.Add(time.Minute)) {
		t.Fatalf("PauseOf = %+v, %v, want a pause ending after the least pause", pause, paused)
	}
}

func TestFailedOfferGivesAFailedPause(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")

	peers.OfferFailed(context.Background(), peer)

	pause, paused := peers.PauseOf(peer)
	want := peereligibility.Pause{
		EndsAt: clk.at.Add(time.Minute),
		Cause:  peereligibility.PauseFailed,
	}
	if !paused || pause != want {
		t.Fatalf("PauseOf = %+v, %v, want %+v", pause, paused, want)
	}
}

func TestTheLatestOfferOutcomeDecidesThePauseCause(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")
	peers.OfferFailed(context.Background(), peer)

	peers.OfferDeclined(context.Background(), peer, 0)

	if pause, _ := peers.PauseOf(peer); pause.Cause != peereligibility.PauseDeclined {
		t.Fatalf("pause cause = %q, want %q", pause.Cause, peereligibility.PauseDeclined)
	}
}

func TestAcceptanceClearsThePause(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Hour, clk)
	peer := yacymodel.WordHash("peer")
	peers.OfferFailed(context.Background(), peer)

	peers.OfferAccepted(peer)

	if pause, paused := peers.PauseOf(peer); paused {
		t.Fatalf("PauseOf = %+v, want no pause after the peer accepted an offer", pause)
	}
}

func TestNoPauseRemainsOnceItEnds(t *testing.T) {
	clk := &clock{at: time.Unix(1000, 0)}
	peers := peersWith(time.Minute, clk)
	peer := yacymodel.WordHash("peer")
	peers.OfferDeclined(context.Background(), peer, 0)

	clk.at = clk.at.Add(time.Minute)

	if pause, paused := peers.PauseOf(peer); paused {
		t.Fatalf("PauseOf = %+v, want no pause once it ends", pause)
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
			LeastPause:                      h.leastPause,
			UnacceptedOffersBeforeExclusion: 3,
			ExclusionDuration:               time.Hour,
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

func (h *exclusionHarness) declineOffers(amountOfDeclinedOffers int) {
	for range amountOfDeclinedOffers {
		h.peers.OfferDeclined(context.Background(), h.peer, 0)
	}
}

func TestDeclinedOffersInARowExcludeThePeerForTheDecline(t *testing.T) {
	h := newExclusionHarness()

	h.declineOffers(3)

	if !h.peers.IsExcluded(h.peer) {
		t.Fatal("IsExcluded = false, want true after declined offers in a row")
	}
	if len(h.counter.exclusionCauses) != 1 ||
		h.counter.exclusionCauses[0] != string(peereligibility.PauseDeclined) {
		t.Fatalf(
			"exclusion causes = %v, want [%s]",
			h.counter.exclusionCauses,
			peereligibility.PauseDeclined,
		)
	}
}

func TestDeclineKeepsTheDeclinedPauseWhenItExcludes(t *testing.T) {
	h := newExclusionHarness()
	h.declineOffers(2)

	h.peers.OfferDeclined(context.Background(), h.peer, time.Hour)

	pause, paused := h.peers.PauseOf(h.peer)
	want := peereligibility.Pause{
		EndsAt: h.clk.at.Add(time.Hour),
		Cause:  peereligibility.PauseDeclined,
	}
	if !paused || pause != want {
		t.Fatalf("PauseOf = %+v, %v, want %+v", pause, paused, want)
	}
}

func TestDeclinedAndFailedOffersCountTogether(t *testing.T) {
	h := newExclusionHarness()
	h.failOffers(2)

	h.declineOffers(1)

	if !h.peers.IsExcluded(h.peer) {
		t.Fatal("IsExcluded = false, want true: a decline counts with failed offers")
	}
	if len(h.counter.exclusionCauses) != 1 ||
		h.counter.exclusionCauses[0] != string(peereligibility.PauseDeclined) {
		t.Fatalf(
			"exclusion causes = %v, want [%s]",
			h.counter.exclusionCauses,
			peereligibility.PauseDeclined,
		)
	}
}

func TestFailedOffersExcludeThePeerForTheFailure(t *testing.T) {
	h := newExclusionHarness()

	h.failOffers(3)

	if len(h.counter.exclusionCauses) != 1 ||
		h.counter.exclusionCauses[0] != string(peereligibility.PauseFailed) {
		t.Fatalf(
			"exclusion causes = %v, want [%s]",
			h.counter.exclusionCauses,
			peereligibility.PauseFailed,
		)
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
