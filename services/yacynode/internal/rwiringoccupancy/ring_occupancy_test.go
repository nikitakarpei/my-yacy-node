package rwiringoccupancy_test

import (
	"context"
	"maps"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/postingcollectiontest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiringoccupancy"
)

const (
	ringPartitionExponent   = 6
	fewerPartitionsExponent = 1
)

type harness struct {
	vault         *vault.Vault
	ringOccupancy rwiringoccupancy.RingOccupancyQuery
	postings      *postingcollectiontest.Postings
}

func openHarness(t *testing.T) harness {
	t.Helper()

	return openHarnessOver(t, memoryvault.OpenEngine(0), ringPartitionExponent)
}

func openHarnessOver(t *testing.T, engine vault.Engine, partitionExponent uint) harness {
	t.Helper()

	v, err := vault.New(engine, nil, nil, wallclock.Clock{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	ringOccupancy, projection, err := rwiringoccupancy.Open(
		v,
		ringPartitionsOf(t, partitionExponent),
	)
	if err != nil {
		t.Fatalf("rwiringoccupancy.Open: %v", err)
	}
	postings, err := postingcollectiontest.Open(context.Background(), v, projection)
	if err != nil {
		t.Fatalf("postingcollectiontest.Open: %v", err)
	}

	return harness{vault: v, ringOccupancy: ringOccupancy, postings: postings}
}

func (h harness) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (h harness) store(t *testing.T, postings ...yacymodel.RWIPosting) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error { return h.postings.Store(tx, postings...) })
}

func (h harness) remove(t *testing.T, postings ...yacymodel.RWIPosting) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error { return h.postings.Remove(tx, postings...) })
}

func (h harness) occupancyOfDHTRingSectors(
	t *testing.T,
) map[yacymodel.DHTRingSector]int {
	t.Helper()

	var occupancyPerSector map[yacymodel.DHTRingSector]int
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		occupancy, err := h.ringOccupancy.OccupancyOfDHTRingSectors(tx)
		occupancyPerSector = occupancy

		return err
	}); err != nil {
		t.Fatalf("OccupancyOfDHTRingSectors: %v", err)
	}

	return occupancyPerSector
}

func ringPartitionsOf(t *testing.T, exponent uint) yacymodel.DHTRingPartitions {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(exponent)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}

	return partitions
}

func postingOf(word, document string) yacymodel.RWIPosting {
	address, err := url.Parse("http://example.com/" + document)
	if err != nil {
		panic(err)
	}

	return yacymodel.RWIPosting{
		WordHash: yacymodel.WordHash(word),
		URLHash:  yacymodel.URLHashOf(address),
		Language: yacymodel.LanguageOfUndeclaredDocument,
		Hits:     1,
	}
}

func sectorOf(t *testing.T, posting yacymodel.RWIPosting) yacymodel.DHTRingSector {
	t.Helper()

	return sectorWithin(t, posting, ringPartitionExponent)
}

func sectorWithin(
	t *testing.T,
	posting yacymodel.RWIPosting,
	partitionExponent uint,
) yacymodel.DHTRingSector {
	t.Helper()

	return yacymodel.DHTRingSectorOf(
		yacymodel.DHTRingPositionOfPosting(posting, ringPartitionsOf(t, partitionExponent)),
	)
}

func occupancyWithin(
	t *testing.T,
	partitionExponent uint,
	postings ...yacymodel.RWIPosting,
) map[yacymodel.DHTRingSector]int {
	t.Helper()

	occupancyPerSector := map[yacymodel.DHTRingSector]int{}
	for _, posting := range postings {
		occupancyPerSector[sectorWithin(t, posting, partitionExponent)]++
	}

	return occupancyPerSector
}

func TestANodeHoldingNoPostingOccupiesNoSector(t *testing.T) {
	h := openHarness(t)

	if occupancy := h.occupancyOfDHTRingSectors(t); len(occupancy) != 0 {
		t.Fatalf("occupancy = %v, want none", occupancy)
	}
}

func TestEachStoredPostingRaisesTheOccupancyOfItsSector(t *testing.T) {
	h := openHarness(t)
	together := []yacymodel.RWIPosting{postingOf("w1", "u1"), postingOf("w2", "u1")}
	apart := postingOf("w1", "u2")

	h.store(t, together[0], together[1], apart)

	want := map[yacymodel.DHTRingSector]int{
		sectorOf(t, together[0]): 2,
		sectorOf(t, apart):       1,
	}
	if got := h.occupancyOfDHTRingSectors(t); !maps.Equal(got, want) {
		t.Fatalf("occupancy = %v, want %v", got, want)
	}
}

func TestEachRemovedPostingLowersTheOccupancyOfItsSector(t *testing.T) {
	h := openHarness(t)
	remaining := postingOf("w1", "u1")
	h.store(t, remaining, postingOf("w2", "u1"))

	h.remove(t, postingOf("w2", "u1"))

	want := map[yacymodel.DHTRingSector]int{sectorOf(t, remaining): 1}
	if got := h.occupancyOfDHTRingSectors(t); !maps.Equal(got, want) {
		t.Fatalf("occupancy = %v, want %v", got, want)
	}
}

func TestTheLastRemovedPostingLeavesItsSectorUnoccupied(t *testing.T) {
	h := openHarness(t)
	h.store(t, postingOf("w1", "u1"))

	h.remove(t, postingOf("w1", "u1"))

	if occupancy := h.occupancyOfDHTRingSectors(t); len(occupancy) != 0 {
		t.Fatalf("occupancy = %v, want none", occupancy)
	}
}

func TestAPostingRemovedAndStoredAgainLeavesTheOccupancyAlone(t *testing.T) {
	h := openHarness(t)
	posting := postingOf("w1", "u1")
	h.store(t, posting)

	h.write(t, func(tx *vault.Txn) error {
		if err := h.postings.Remove(tx, posting); err != nil {
			return err
		}

		return h.postings.Store(tx, posting)
	})

	want := map[yacymodel.DHTRingSector]int{sectorOf(t, posting): 1}
	if got := h.occupancyOfDHTRingSectors(t); !maps.Equal(got, want) {
		t.Fatalf("occupancy = %v, want %v", got, want)
	}
}

func TestAPostingStoredAgainLeavesTheOccupancyAlone(t *testing.T) {
	h := openHarness(t)
	posting := postingOf("w1", "u1")
	storedAgain := posting
	storedAgain.Hits = 2

	h.store(t, posting)
	h.store(t, storedAgain)

	want := map[yacymodel.DHTRingSector]int{sectorOf(t, posting): 1}
	if got := h.occupancyOfDHTRingSectors(t); !maps.Equal(got, want) {
		t.Fatalf("occupancy = %v, want %v", got, want)
	}
}

func TestAChangedPartitionCountRecountsTheSectorsAtOpen(t *testing.T) {
	engine := memoryvault.OpenEngine(0)
	postings := []yacymodel.RWIPosting{
		postingOf("w1", "u1"),
		postingOf("w2", "u2"),
		postingOf("w3", "u3"),
		postingOf("w4", "u4"),
	}
	openHarnessOver(t, engine, ringPartitionExponent).store(t, postings...)
	want := occupancyWithin(t, fewerPartitionsExponent, postings...)
	if maps.Equal(want, occupancyWithin(t, ringPartitionExponent, postings...)) {
		t.Fatal(
			"both partition counts put the postings in the same sectors, want postings that move",
		)
	}

	h := openHarnessOver(t, engine, fewerPartitionsExponent)

	if got := h.occupancyOfDHTRingSectors(t); !maps.Equal(got, want) {
		t.Fatalf("occupancy = %v, want %v", got, want)
	}
}
