package postinghandoff

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultengines/memory"
)

const handoffRedundancy = 1

type fakePostingPurger struct {
	purgedPostings []yacymodel.RWIPosting
}

func (f *fakePostingPurger) PurgePosting(
	_ *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (bool, error) {
	f.purgedPostings = append(f.purgedPostings, yacymodel.RWIPosting{WordHash: word, URLHash: url})

	return true, nil
}

type fakeReachability struct {
	reachablePeers []yacymodel.Hash
}

func (f fakeReachability) IsReachable(_ context.Context, peer yacymodel.Hash) bool {
	for _, reachable := range f.reachablePeers {
		if reachable == peer {
			return true
		}
	}

	return false
}

func urlHash() yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash("u1").String())
	if err != nil {
		panic(err)
	}

	return hash
}

func thisNodeFartherThanEveryPeer() yacymodel.Hash { return yacymodel.WordHash("self5") }

func openHandoff(
	t *testing.T,
	reachability Reachability,
	purger rwipostings.PostingPurger,
	redundancy int,
) (*vault.Vault, *postingofferschedule.Schedule, *postingreplicas.Replicas, *Handoff) {
	t.Helper()

	v, err := memory.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	schedule, err := postingofferschedule.Open(
		v,
		func() time.Time { return time.Unix(1000, 0) },
		discardedScheduleObservations{},
	)
	if err != nil {
		t.Fatalf("postingofferschedule.Open: %v", err)
	}
	replicas, err := postingreplicas.Open(v, schedule)
	if err != nil {
		t.Fatalf("postingreplicas.Open: %v", err)
	}
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}

	handoff := New(
		replicas,
		purger,
		reachability,
		partitions,
		thisNodeFartherThanEveryPeer(),
		redundancy,
	)

	return v, schedule, replicas, handoff
}

func store(
	t *testing.T,
	v *vault.Vault,
	schedule *postingofferschedule.Schedule,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return schedule.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
}

func recordAccepted(
	t *testing.T,
	v *vault.Vault,
	replicas *postingreplicas.Replicas,
	peer yacymodel.Hash,
	postings ...yacymodel.RWIPosting,
) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return replicas.RecordAccepted(tx, peer, postings)
	}); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}
}

func handOff(
	t *testing.T,
	v *vault.Vault,
	handoff *Handoff,
	posting yacymodel.RWIPosting,
) int {
	t.Helper()

	var handedOffPostings int
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		var err error
		handedOffPostings, err = handoff.HandOffPostingsHeldByCloserPeers(
			context.Background(), tx, []yacymodel.RWIPosting{posting},
		)

		return err
	}); err != nil {
		t.Fatalf("HandOffPostingsHeldByCloserPeers: %v", err)
	}

	return handedOffPostings
}

func TestPostingHandedOffOnceEnoughReachableCloserHoldersExist(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	closerPeer := yacymodel.WordHash("peer")
	posting := yacymodel.RWIPosting{WordHash: word, URLHash: url}
	purger := &fakePostingPurger{}
	v, schedule, replicas, handoff := openHandoff(
		t,
		fakeReachability{reachablePeers: []yacymodel.Hash{closerPeer}},
		purger,
		handoffRedundancy,
	)

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, closerPeer, posting)

	if handedOffPostings := handOff(t, v, handoff, posting); handedOffPostings != 1 {
		t.Fatalf(
			"handed off = %d, want 1: redundancy is met by a closer peer", handedOffPostings,
		)
	}
	if len(purger.purgedPostings) != 1 {
		t.Fatalf("purged = %+v, want the handed-off posting", purger.purgedPostings)
	}
}

func TestPostingKeptBelowRedundancy(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	closerPeer := yacymodel.WordHash("peer")
	posting := yacymodel.RWIPosting{WordHash: word, URLHash: url}
	purger := &fakePostingPurger{}
	v, schedule, replicas, handoff := openHandoff(
		t, fakeReachability{reachablePeers: []yacymodel.Hash{closerPeer}}, purger, 2,
	)

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, closerPeer, posting)

	if handedOffPostings := handOff(t, v, handoff, posting); handedOffPostings != 0 {
		t.Fatalf(
			"handed off = %d, want 0: only one of two owed peers holds it", handedOffPostings,
		)
	}
	if len(purger.purgedPostings) != 0 {
		t.Fatalf("purged = %+v, want none", purger.purgedPostings)
	}
}

func TestPostingKeptWhileTheCloserHolderIsUnreachable(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	closerPeer := yacymodel.WordHash("peer")
	posting := yacymodel.RWIPosting{WordHash: word, URLHash: url}
	purger := &fakePostingPurger{}
	v, schedule, replicas, handoff := openHandoff(
		t, fakeReachability{}, purger, handoffRedundancy,
	)

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, closerPeer, posting)

	if handedOffPostings := handOff(t, v, handoff, posting); handedOffPostings != 0 {
		t.Fatalf("handed off = %d, want 0: the holder is unreachable", handedOffPostings)
	}
}

type discardedScheduleObservations struct{}

func (discardedScheduleObservations) ObserveScheduledPostings(int) {}

func (discardedScheduleObservations) ObserveLongestOfferLateness(time.Duration) {}
