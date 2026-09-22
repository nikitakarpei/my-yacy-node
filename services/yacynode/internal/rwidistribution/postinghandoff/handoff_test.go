package postinghandoff_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postinghandoff"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
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

type handoffHarness struct {
	vault    *vault.Vault
	schedule *postingofferschedule.Schedule
	replicas *postingreplicas.Replicas
	purger   *fakePostingPurger
	handoff  *postinghandoff.Handoff
}

func openHandoff(
	t *testing.T,
	reachability postinghandoff.Reachability,
	redundancy int,
) handoffHarness {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}
	schedule, err := postingofferschedule.Open(
		v,
		partitions,
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

	purger := &fakePostingPurger{}
	handoff := postinghandoff.New(replicas, purger, reachability, postinghandoff.Config{
		Partitions: partitions,
		Self:       thisNodeFartherThanEveryPeer(),
		Redundancy: redundancy,
	})

	return handoffHarness{
		vault:    v,
		schedule: schedule,
		replicas: replicas,
		purger:   purger,
		handoff:  handoff,
	}
}

func (h handoffHarness) store(t *testing.T, word yacymodel.Hash, url yacymodel.URLHash) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.schedule.PostingStored(tx, yacymodel.RWIPosting{WordHash: word, URLHash: url})
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
}

func (h handoffHarness) recordAccepted(
	t *testing.T,
	peer yacymodel.Hash,
	postings ...yacymodel.RWIPosting,
) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.replicas.RecordAccepted(tx, peer, postings)
	}); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}
}

func (h handoffHarness) handOff(t *testing.T, posting yacymodel.RWIPosting) int {
	t.Helper()

	var handedOffPostings int
	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		var err error
		handedOffPostings, err = h.handoff.HandOffPostingsHeldByCloserPeers(
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
	h := openHandoff(
		t, fakeReachability{reachablePeers: []yacymodel.Hash{closerPeer}}, handoffRedundancy,
	)

	h.store(t, word, url)
	h.recordAccepted(t, closerPeer, posting)

	if handedOffPostings := h.handOff(t, posting); handedOffPostings != 1 {
		t.Fatalf(
			"handed off = %d, want 1: redundancy is met by a closer peer", handedOffPostings,
		)
	}
	if len(h.purger.purgedPostings) != 1 {
		t.Fatalf("purged = %+v, want the handed-off posting", h.purger.purgedPostings)
	}
}

func TestPostingKeptBelowRedundancy(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	closerPeer := yacymodel.WordHash("peer")
	posting := yacymodel.RWIPosting{WordHash: word, URLHash: url}
	h := openHandoff(t, fakeReachability{reachablePeers: []yacymodel.Hash{closerPeer}}, 2)

	h.store(t, word, url)
	h.recordAccepted(t, closerPeer, posting)

	if handedOffPostings := h.handOff(t, posting); handedOffPostings != 0 {
		t.Fatalf(
			"handed off = %d, want 0: only one of two owed peers holds it", handedOffPostings,
		)
	}
	if len(h.purger.purgedPostings) != 0 {
		t.Fatalf("purged = %+v, want none", h.purger.purgedPostings)
	}
}

func TestPostingKeptWhileTheCloserHolderIsUnreachable(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	closerPeer := yacymodel.WordHash("peer")
	posting := yacymodel.RWIPosting{WordHash: word, URLHash: url}
	h := openHandoff(t, fakeReachability{}, handoffRedundancy)

	h.store(t, word, url)
	h.recordAccepted(t, closerPeer, posting)

	if handedOffPostings := h.handOff(t, posting); handedOffPostings != 0 {
		t.Fatalf("handed off = %d, want 0: the holder is unreachable", handedOffPostings)
	}
}

func TestKeepEveryPostingHandsOffNothing(t *testing.T) {
	handedOffPostings, err := postinghandoff.KeepEveryPosting{}.HandOffPostingsHeldByCloserPeers(
		context.Background(),
		nil,
		[]yacymodel.RWIPosting{{WordHash: yacymodel.WordHash("w1"), URLHash: urlHash()}},
	)
	if err != nil {
		t.Fatalf("HandOffPostingsHeldByCloserPeers: %v", err)
	}
	if handedOffPostings != 0 {
		t.Fatalf("handed off = %d, want 0: every posting is kept", handedOffPostings)
	}
}

type discardedScheduleObservations struct{}

func (discardedScheduleObservations) ObserveScheduledPostings(string, int) {}

func (discardedScheduleObservations) ObserveLongestOfferLateness(string, time.Duration) {}
