package rwiringoccupancy_test

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiringoccupancy"
)

const (
	ringPartitionExponent   = 6
	fewerPartitionsExponent = 1
)

type harness struct {
	vault         *vault.Vault
	ringOccupancy rwiringoccupancy.RingOccupancyQuery
	writer        rwipostings.PostingWriter
	purger        rwipostings.PostingPurger
}

func openHarness(t *testing.T) harness {
	t.Helper()

	return openHarnessOver(t, memoryvault.OpenEngine(0), ringPartitionExponent)
}

func openHarnessOver(t *testing.T, engine vault.Engine, partitionExponent uint) harness {
	t.Helper()

	v, err := vault.New(engine, nil, wallclock.Clock{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	partitions := ringPartitionsOf(t, partitionExponent)
	ringOccupancy, projection, err := rwiringoccupancy.Open(v, partitions)
	if err != nil {
		t.Fatalf("rwiringoccupancy.Open: %v", err)
	}
	_, writer, purger, err := rwipostings.Open(
		context.Background(),
		v,
		partitions,
		nil,
		nil,
		[]vault.Projection[yacymodel.RWIPosting]{projection},
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}

	return harness{vault: v, ringOccupancy: ringOccupancy, writer: writer, purger: purger}
}

func (h harness) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (h harness) store(t *testing.T, postings ...yacymodel.RWIPosting) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		for _, posting := range postings {
			if err := h.writer.Write(tx, posting); err != nil {
				return fmt.Errorf("store posting: %w", err)
			}
		}

		return nil
	})
}

func (h harness) purge(t *testing.T, postings ...yacymodel.RWIPosting) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		for _, posting := range postings {
			if _, err := h.purger.PurgePosting(tx, posting.WordHash, posting.URLHash); err != nil {
				return fmt.Errorf("purge posting: %w", err)
			}
		}

		return nil
	})
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
		URLHash:  yacymodel.URLNormalformOf(address).Hash(),
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

func TestEachPurgedPostingLowersTheOccupancyOfItsSector(t *testing.T) {
	h := openHarness(t)
	remaining := postingOf("w1", "u1")
	h.store(t, remaining, postingOf("w2", "u1"))

	h.purge(t, postingOf("w2", "u1"))

	want := map[yacymodel.DHTRingSector]int{sectorOf(t, remaining): 1}
	if got := h.occupancyOfDHTRingSectors(t); !maps.Equal(got, want) {
		t.Fatalf("occupancy = %v, want %v", got, want)
	}
}

func TestTheLastPurgedPostingLeavesItsSectorUnoccupied(t *testing.T) {
	h := openHarness(t)
	h.store(t, postingOf("w1", "u1"))

	h.purge(t, postingOf("w1", "u1"))

	if occupancy := h.occupancyOfDHTRingSectors(t); len(occupancy) != 0 {
		t.Fatalf("occupancy = %v, want none", occupancy)
	}
}

func TestAPostingPurgedAndStoredAgainLeavesTheOccupancyAlone(t *testing.T) {
	h := openHarness(t)
	posting := postingOf("w1", "u1")
	h.store(t, posting)

	h.write(t, func(tx *vault.Txn) error {
		if _, err := h.purger.PurgePosting(tx, posting.WordHash, posting.URLHash); err != nil {
			return err
		}

		return h.writer.Write(tx, posting)
	})

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
