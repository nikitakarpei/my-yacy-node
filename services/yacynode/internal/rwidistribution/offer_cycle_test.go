package rwidistribution

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type fakeCourier struct {
	outcomes map[yacymodel.Hash]offerOutcome
	offered  []postingOffer
	onOffer  func()
}

func (f *fakeCourier) Offer(_ context.Context, offer postingOffer) offerOutcome {
	f.offered = append(f.offered, offer)
	if f.onOffer != nil {
		f.onOffer()
	}

	return f.outcomes[offer.Peer.Hash]
}

const offerCycleRedundancy = 1

func openOfferCycle(
	t *testing.T,
	now func() time.Time,
	postings map[yacymodel.Hash]yacymodel.RWIPosting,
	responsible []yacymodel.Seed,
) (*offerSchedule, *replicaLedger, *fakeCourier, *fakeOfferObserver, *offerCycle) {
	t.Helper()

	schedule, ledger, planner, observer := openOfferPlanner(t, now, postings, responsible)
	courier := &fakeCourier{outcomes: make(map[yacymodel.Hash]offerOutcome)}

	cycle := &offerCycle{
		planner:          planner,
		courier:          courier,
		schedule:         schedule,
		ledger:           ledger,
		observer:         observer,
		now:              now,
		postingsPerCycle: 10,
		cycleInterval:    time.Minute,
		refreshInterval:  time.Hour,
		retryInterval:    time.Minute,
		redundancy:       offerCycleRedundancy,
	}

	return schedule, ledger, courier, observer, cycle
}

func TestOfferCycleRunOffersOnceThenStops(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, _, courier, observer, cycle := openOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	courier.outcomes[peer] = offerOutcome{Accepted: true}

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	courier.onOffer = cancel
	cycle.Run(ctx)

	if len(courier.offered) != 1 {
		t.Fatalf("offered = %v, want a single offer from the initial run", courier.offered)
	}
	if observer.drained != 1 {
		t.Fatalf("drained = %v, want 1", observer.drained)
	}
}

func TestOfferCycleDropsStaleReplicaFromLedger(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	stalePeer, fresh := yacymodel.WordHash("stale"), yacymodel.WordHash("fresh")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, ledger, courier, observer, cycle := openOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(fresh)},
	)
	courier.outcomes[fresh] = offerOutcome{Accepted: true}

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	if err := ledger.RecordAccepted(context.Background(), word, url, stalePeer); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	cycle.offerOnce(context.Background())

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	for _, replica := range replicas {
		if replica == stalePeer {
			t.Fatalf("replicas = %v, want %v dropped", replicas, stalePeer)
		}
	}
	if observer.prunes != 1 {
		t.Fatalf("prunes = %v, want 1", observer.prunes)
	}
}

func TestOfferCycleReschedulesAcceptedPostingAtRefreshInterval(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, ledger, courier, _, cycle := openOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	courier.outcomes[peer] = offerOutcome{Accepted: true}

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	if err := ledger.RecordAccepted(context.Background(), word, url, peer); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	cycle.offerOnce(context.Background())

	due, err := schedule.DueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("DueBatch: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due right after a fully replicated posting", due)
	}
}

func TestOfferCycleRetriesRejectedPostingAtRetryInterval(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, _, courier, _, cycle := openOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	courier.outcomes[peer] = offerOutcome{}

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	cycle.offerOnce(context.Background())

	due, err := schedule.DueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("DueBatch: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due immediately after a rejected offer", due)
	}

	future := now.Add(cycle.retryInterval + time.Second)
	schedule.now = func() time.Time { return future }
	due, err = schedule.DueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("DueBatch after retry interval: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the retry interval has elapsed", due)
	}
}

func TestOfferCycleHonoursCourierRetryAfter(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, _, courier, _, cycle := openOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	courier.outcomes[peer] = offerOutcome{RetryAfter: 5 * time.Minute}

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	cycle.offerOnce(context.Background())

	afterCycleRetry := now.Add(cycle.retryInterval + time.Second)
	schedule.now = func() time.Time { return afterCycleRetry }
	due, err := schedule.DueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("DueBatch: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due before the peer's pause elapses", due)
	}

	afterPause := now.Add(5*time.Minute + time.Second)
	schedule.now = func() time.Time { return afterPause }
	due, err = schedule.DueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("DueBatch after pause: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the peer's pause has elapsed", due)
	}
}

func TestOfferCycleReschedulesUnofferedPostingAtRetryInterval(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, _, courier, _, cycle := openOfferCycle(
		t,
		func() time.Time { return now },
		postings,
		nil,
	)

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	cycle.offerOnce(context.Background())

	if len(courier.offered) != 0 {
		t.Fatalf("offered = %v, want no offers for an unoffered posting", courier.offered)
	}

	due, err := schedule.DueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("DueBatch: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due immediately after stalling", due)
	}

	future := now.Add(cycle.retryInterval + time.Second)
	schedule.now = func() time.Time { return future }
	due, err = schedule.DueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("DueBatch after retry interval: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the retry interval has elapsed", due)
	}
}

func TestOfferCycleReschedulesAlreadySatisfiedPostingAtRefreshInterval(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, ledger, courier, _, cycle := openOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	if err := ledger.RecordAccepted(context.Background(), word, url, peer); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	cycle.offerOnce(context.Background())

	if len(courier.offered) != 0 {
		t.Fatalf("offered = %v, want no offers for an already replicated posting", courier.offered)
	}

	future := now.Add(cycle.refreshInterval - time.Second)
	schedule.now = func() time.Time { return future }
	due, err := schedule.DueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("DueBatch: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due before the refresh interval elapses", due)
	}
}
