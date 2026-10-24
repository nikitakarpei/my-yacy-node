package peereligibility_test

import (
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/peereligibility"
)

type sendHarness struct {
	observer *fakeObserver
	peers    *peereligibility.Peers
	peer     yacymodel.Hash
}

func newSendHarness(workInFlight time.Duration) *sendHarness {
	h := &sendHarness{observer: &fakeObserver{}, peer: yacymodel.WordHash("peer")}
	h.peers = peereligibility.New(
		peereligibility.Config{
			LeastPause:                      time.Minute,
			UnacceptedOffersBeforeExclusion: 3,
			ExclusionDuration:               time.Hour,
			WorkInFlight:                    workInFlight,
		},
		(&clock{at: time.Unix(1000, 0)}).now,
		h.observer,
	)

	return h
}

func (h *sendHarness) endSendsAfter(sendTimes ...time.Duration) {
	for _, sendTime := range sendTimes {
		h.peers.SendStarted(h.peer)
		h.peers.SendEnded(h.peer, sendTime)
	}
}

func (h *sendHarness) startSends(amountOfSends int) {
	for range amountOfSends {
		h.peers.SendStarted(h.peer)
	}
}

func (h *sendHarness) assertBusyFrom(t *testing.T, amountOfSendsAllowed int) {
	t.Helper()

	h.startSends(amountOfSendsAllowed - 1)
	if h.peers.IsBusy(h.peer) {
		t.Fatalf("IsBusy = true, want false with %d sends in flight", amountOfSendsAllowed-1)
	}
	h.startSends(1)
	if !h.peers.IsBusy(h.peer) {
		t.Fatalf("IsBusy = false, want true with %d sends in flight", amountOfSendsAllowed)
	}
}

func TestAPeerWithoutASendTimeIsNeverBusy(t *testing.T) {
	h := newSendHarness(time.Second)

	h.startSends(10)

	if h.peers.IsBusy(h.peer) {
		t.Fatal("IsBusy = true, want false for a peer that never ended a send")
	}
}

func TestASlowPeerIsBusyWithOneSendInFlight(t *testing.T) {
	h := newSendHarness(time.Second)
	h.endSendsAfter(2 * time.Second)

	h.assertBusyFrom(t, 1)
}

func TestASlowPeerIsNoLongerBusyOnceItsSendEnds(t *testing.T) {
	h := newSendHarness(time.Second)
	h.endSendsAfter(2 * time.Second)
	h.startSends(1)

	h.peers.SendEnded(h.peer, 2*time.Second)

	if h.peers.IsBusy(h.peer) {
		t.Fatal("IsBusy = true, want false once its send ended")
	}
}

func TestAFastPeerTakesTheSendsItsSendTimeFitsIntoTheWorkInFlight(t *testing.T) {
	h := newSendHarness(time.Second)
	h.endSendsAfter(250 * time.Millisecond)

	h.assertBusyFrom(t, 4)
}

func TestEachSendTimeWeighsAFifthInTheUsualSendTime(t *testing.T) {
	h := newSendHarness(3 * time.Second)

	h.endSendsAfter(time.Second, 3*time.Second)

	h.assertBusyFrom(t, 2)
}

func TestAPeerThatSpeedsUpTakesMoreSendsAgain(t *testing.T) {
	h := newSendHarness(time.Second)
	h.endSendsAfter(2 * time.Second)

	h.endSendsAfter(slices.Repeat([]time.Duration{100 * time.Millisecond}, 20)...)

	h.assertBusyFrom(t, 8)
}

func TestThePeersAtTheirSendLimitAreReported(t *testing.T) {
	h := newSendHarness(time.Second)
	h.endSendsAfter(2 * time.Second)

	h.startSends(1)
	if h.observer.peersAtSendLimit != 1 {
		t.Fatalf("peers at send limit = %d, want 1 while its send is in flight",
			h.observer.peersAtSendLimit)
	}
	h.peers.SendEnded(h.peer, 2*time.Second)
	if h.observer.peersAtSendLimit != 0 {
		t.Fatalf("peers at send limit = %d, want 0 once its send ended",
			h.observer.peersAtSendLimit)
	}
}

func TestEachEndedSendReportsItsSendTime(t *testing.T) {
	h := newSendHarness(time.Second)

	h.endSendsAfter(2*time.Second, 250*time.Millisecond)

	want := []time.Duration{2 * time.Second, 250 * time.Millisecond}
	if !slices.Equal(h.observer.sendTimes, want) {
		t.Fatalf("send times = %v, want %v", h.observer.sendTimes, want)
	}
}
