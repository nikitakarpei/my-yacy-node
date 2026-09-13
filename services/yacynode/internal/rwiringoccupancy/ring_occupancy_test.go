package rwiringoccupancy_test

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiringoccupancy"
)

const ringPartitionExponent = 6

type harness struct {
	vault         *vault.Vault
	ringOccupancy rwiringoccupancy.RingOccupancyProjection
}

func openHarness(t *testing.T) harness {
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

	ringOccupancy, err := rwiringoccupancy.Open(v, ringPartitions(t))
	if err != nil {
		t.Fatalf("rwiringoccupancy.Open: %v", err)
	}

	return harness{vault: v, ringOccupancy: ringOccupancy}
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
			if err := h.ringOccupancy.PostingStored(tx, posting); err != nil {
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
			if err := h.ringOccupancy.PostingPurged(tx, posting); err != nil {
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

func ringPartitions(t *testing.T) yacymodel.DHTRingPartitions {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(ringPartitionExponent)
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

	return yacymodel.DHTRingSectorOf(
		yacymodel.DHTRingPositionOfPosting(posting, ringPartitions(t)),
	)
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

	h.purge(t, posting)
	h.store(t, posting)

	want := map[yacymodel.DHTRingSector]int{sectorOf(t, posting): 1}
	if got := h.occupancyOfDHTRingSectors(t); !maps.Equal(got, want) {
		t.Fatalf("occupancy = %v, want %v", got, want)
	}
}
