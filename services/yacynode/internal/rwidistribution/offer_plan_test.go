package rwidistribution

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
)

type fakePostingIndex struct {
	postings map[yacymodel.Hash]yacymodel.RWIPosting
}

func (f fakePostingIndex) RWICount(context.Context) (int, error) { return len(f.postings), nil }

func (f fakePostingIndex) Posting(
	_ context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	entry, found := f.postings[postingIdentity(word, url)]

	return entry, found, nil
}

func (f fakePostingIndex) ScanWord(
	context.Context,
	yacymodel.Hash,
	func(yacymodel.RWIPosting) (bool, error),
) error {
	return nil
}

func postingIdentity(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.Hash {
	return yacymodel.WordHash(word.String() + url.String())
}

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func fakePosting(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{WordHash: word, URLHash: url}
}

type fakeRoster struct {
	responsible []yacymodel.Seed
	reachable   []yacymodel.Seed
}

func (fakeRoster) Discover(context.Context, ...yacymodel.Seed)         {}
func (fakeRoster) ConfirmReachable(context.Context, yacymodel.Hash)    {}
func (fakeRoster) ConfirmUnreachable(context.Context, yacymodel.Hash)  {}
func (fakeRoster) FreshestPeers(context.Context, int) []yacymodel.Seed { return nil }

func (f fakeRoster) ReachablePeers(context.Context) []yacymodel.Seed {
	return f.reachable
}

func (f fakeRoster) PeersResponsibleFor(
	context.Context,
	yacymodel.DHTPosition,
	int,
) []yacymodel.Seed {
	return f.responsible
}

func seed(hash yacymodel.Hash) yacymodel.Seed {
	return yacymodel.Seed{Hash: hash}
}

const offerPlannerRedundancy = 1

func openOfferPlanner(
	t *testing.T,
	now func() time.Time,
	postings map[yacymodel.Hash]yacymodel.RWIPosting,
	responsible []yacymodel.Seed,
) (*offerSchedule, *replicaLedger, *offerPlanner, *fakeOfferObserver) {
	t.Helper()

	v, err := memvault.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	schedule, err := openOfferSchedule(v, now)
	if err != nil {
		t.Fatalf("openOfferSchedule: %v", err)
	}
	ledger, err := openReplicaLedger(v)
	if err != nil {
		t.Fatalf("openReplicaLedger: %v", err)
	}
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}

	observer := newFakeOfferObserver()
	planner := &offerPlanner{
		schedule:   schedule,
		ledger:     ledger,
		postings:   fakePostingIndex{postings: postings},
		roster:     fakeRoster{responsible: responsible, reachable: responsible},
		observer:   observer,
		partitions: partitions,
		redundancy: offerPlannerRedundancy,
	}

	return schedule, ledger, planner, observer
}

func TestPlanOffersDuePostingToResponsiblePeers(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, _, planner, _ := openOfferPlanner(
		t,
		func() time.Time { return now },
		postings,
		[]yacymodel.Seed{seed(peer)},
	)

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	plan, err := planner.Plan(context.Background(), 10)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Offers) != 1 || plan.Offers[0].Peer.Hash != peer ||
		len(plan.Offers[0].Postings) != 1 {
		t.Fatalf("plan.Offers = %+v, want one offer to %v", plan.Offers, peer)
	}
	if len(plan.Replicated) != 0 || len(plan.Unoffered) != 0 {
		t.Fatalf("plan = %+v, want no replicated or unoffered entries", plan)
	}
}

func TestPlanSkipsFullyReplicatedPosting(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, ledger, planner, _ := openOfferPlanner(
		t,
		func() time.Time { return now },
		postings,
		[]yacymodel.Seed{seed(peer)},
	)

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	if err := ledger.RecordAccepted(context.Background(), word, url, peer); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	plan, err := planner.Plan(context.Background(), 10)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Offers) != 0 {
		t.Fatalf("plan.Offers = %+v, want none", plan.Offers)
	}
	if len(plan.Replicated) != 1 || plan.Replicated[0].Word != word {
		t.Fatalf("plan.Replicated = %+v, want [word]", plan.Replicated)
	}
}

func TestPlanLeavesPostingUnofferedWithNoResponsiblePeers(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, _, planner, _ := openOfferPlanner(t, func() time.Time { return now }, postings, nil)

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	plan, err := planner.Plan(context.Background(), 10)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Offers) != 0 || len(plan.Replicated) != 0 {
		t.Fatalf("plan = %+v, want only an unoffered entry", plan)
	}
	if len(plan.Unoffered) != 1 || plan.Unoffered[0].Word != word {
		t.Fatalf("plan.Unoffered = %+v, want [word]", plan.Unoffered)
	}
}

func TestPlanSkipsPostingRemovedSinceScheduling(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	schedule, _, planner, _ := openOfferPlanner(
		t,
		func() time.Time { return now },
		nil,
		[]yacymodel.Seed{seed(peer)},
	)

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	plan, err := planner.Plan(context.Background(), 10)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Offers) != 0 || len(plan.Replicated) != 0 || len(plan.Unoffered) != 0 {
		t.Fatalf("plan = %+v, want empty plan for a posting missing from the index", plan)
	}
}

func TestPlanSkipsCycleWhenTooFewPeersReachable(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, _, planner, observer := openOfferPlanner(
		t,
		func() time.Time { return now },
		postings,
		[]yacymodel.Seed{seed(peer)},
	)
	planner.minReachablePeers = 2

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	plan, err := planner.Plan(context.Background(), 10)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Offers) != 0 || len(plan.Replicated) != 0 || len(plan.Unoffered) != 0 ||
		plan.Drained != 0 {
		t.Fatalf("plan = %+v, want an empty plan while below the reachable-peer floor", plan)
	}
	if observer.cyclesSkipped != 1 {
		t.Fatalf("cyclesSkipped = %v, want 1", observer.cyclesSkipped)
	}
}

func TestPlanReportsStaleReplicaWithoutDroppingIt(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	stalePeer, fresh := yacymodel.WordHash("stale"), yacymodel.WordHash("fresh")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, ledger, planner, observer := openOfferPlanner(
		t,
		func() time.Time { return now },
		postings,
		[]yacymodel.Seed{seed(fresh)},
	)

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	if err := ledger.RecordAccepted(context.Background(), word, url, stalePeer); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	plan, err := planner.Plan(context.Background(), 10)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Offers) != 1 || plan.Offers[0].Peer.Hash != fresh {
		t.Fatalf(
			"plan.Offers = %+v, want an offer to %v regardless of stale peer %v",
			plan.Offers,
			fresh,
			stalePeer,
		)
	}
	if len(plan.StaleReplicas) != 1 || plan.StaleReplicas[0].Posting.Word != word ||
		plan.StaleReplicas[0].Posting.URL != url || len(plan.StaleReplicas[0].Peers) != 1 ||
		plan.StaleReplicas[0].Peers[0] != stalePeer {
		t.Fatalf("plan.StaleReplicas = %+v, want one entry for %v", plan.StaleReplicas, stalePeer)
	}
	if observer.prunes != 0 {
		t.Fatalf("prunes = %v, want 0 since Plan must not mutate the ledger", observer.prunes)
	}

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 1 || replicas[0] != stalePeer {
		t.Fatalf("replicas = %v, want [%v] unchanged by Plan", replicas, stalePeer)
	}
}
