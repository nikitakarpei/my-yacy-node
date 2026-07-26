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
	word, url yacymodel.Hash,
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

func postingIdentity(word, url yacymodel.Hash) yacymodel.Hash {
	return yacymodel.WordHash(word.String() + url.String())
}

func fakePosting(word, url yacymodel.Hash) yacymodel.RWIPosting {
	urlHash, err := yacymodel.ParseURLHash(url.String())
	if err != nil {
		panic(err)
	}

	return yacymodel.RWIPosting{WordHash: word, URLHash: urlHash}
}

type fakeRoster struct {
	responsible []yacymodel.Seed
}

func (fakeRoster) Discover(context.Context, ...yacymodel.Seed)         {}
func (fakeRoster) ConfirmReachable(context.Context, yacymodel.Hash)    {}
func (fakeRoster) ConfirmUnreachable(context.Context, yacymodel.Hash)  {}
func (fakeRoster) FreshestPeers(context.Context, int) []yacymodel.Seed { return nil }
func (fakeRoster) ReachablePeers(context.Context) []yacymodel.Seed     { return nil }

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

const batchBuilderRedundancy = 1

func openBatchBuilder(
	t *testing.T,
	now func() time.Time,
	postings map[yacymodel.Hash]yacymodel.RWIPosting,
	responsible []yacymodel.Seed,
) (*offerSchedule, *replicaLedger, *batchBuilder) {
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

	builder := &batchBuilder{
		schedule:   schedule,
		ledger:     ledger,
		postings:   fakePostingIndex{postings: postings},
		roster:     fakeRoster{responsible: responsible},
		partitions: partitions,
		redundancy: batchBuilderRedundancy,
	}

	return schedule, ledger, builder
}

func TestBuildOffersDuePostingToResponsiblePeers(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, _, builder := openBatchBuilder(
		t,
		func() time.Time { return now },
		postings,
		[]yacymodel.Seed{seed(peer)},
	)

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	plan, err := builder.Build(context.Background(), 10)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(plan.Offers) != 1 || plan.Offers[0].Peer.Hash != peer ||
		len(plan.Offers[0].Postings) != 1 {
		t.Fatalf("plan.Offers = %+v, want one offer to %v", plan.Offers, peer)
	}
	if len(plan.Satisfied) != 0 || len(plan.Stalled) != 0 {
		t.Fatalf("plan = %+v, want no satisfied or stalled entries", plan)
	}
}

func TestBuildSkipsPostingAlreadySatisfied(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, ledger, builder := openBatchBuilder(
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

	plan, err := builder.Build(context.Background(), 10)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(plan.Offers) != 0 {
		t.Fatalf("plan.Offers = %+v, want none", plan.Offers)
	}
	if len(plan.Satisfied) != 1 || plan.Satisfied[0].Word != word {
		t.Fatalf("plan.Satisfied = %+v, want [word]", plan.Satisfied)
	}
}

func TestBuildStallsPostingWithNoResponsiblePeers(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, _, builder := openBatchBuilder(t, func() time.Time { return now }, postings, nil)

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	plan, err := builder.Build(context.Background(), 10)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(plan.Offers) != 0 || len(plan.Satisfied) != 0 {
		t.Fatalf("plan = %+v, want only a stalled entry", plan)
	}
	if len(plan.Stalled) != 1 || plan.Stalled[0].Word != word {
		t.Fatalf("plan.Stalled = %+v, want [word]", plan.Stalled)
	}
}

func TestBuildSkipsPostingRemovedSinceScheduling(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	peer := yacymodel.WordHash("peer")
	schedule, _, builder := openBatchBuilder(
		t,
		func() time.Time { return now },
		nil,
		[]yacymodel.Seed{seed(peer)},
	)

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	plan, err := builder.Build(context.Background(), 10)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(plan.Offers) != 0 || len(plan.Satisfied) != 0 || len(plan.Stalled) != 0 {
		t.Fatalf("plan = %+v, want empty plan for a posting missing from the index", plan)
	}
}

func TestBuildRePrunesLedgerWhenPeerNoLongerResponsible(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	stale, fresh := yacymodel.WordHash("stale"), yacymodel.WordHash("fresh")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}
	schedule, ledger, builder := openBatchBuilder(
		t,
		func() time.Time { return now },
		postings,
		[]yacymodel.Seed{seed(fresh)},
	)

	if err := schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	if err := ledger.RecordAccepted(context.Background(), word, url, stale); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	plan, err := builder.Build(context.Background(), 10)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(plan.Offers) != 1 || plan.Offers[0].Peer.Hash != fresh {
		t.Fatalf(
			"plan.Offers = %+v, want an offer to %v after pruning %v",
			plan.Offers,
			fresh,
			stale,
		)
	}
}
