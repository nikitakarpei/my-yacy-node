package rwipostingsectoramount_test

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingsectoramount"
)

const ringPartitionExponent = 6

type harness struct {
	vault         *vault.Vault
	sectorAmounts rwipostingsectoramount.PostingSectorAmountProjection
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

	sectorAmounts, err := rwipostingsectoramount.Open(v, ringPartitions(t))
	if err != nil {
		t.Fatalf("rwipostingsectoramount.Open: %v", err)
	}

	return harness{vault: v, sectorAmounts: sectorAmounts}
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
			if err := h.sectorAmounts.PostingStored(tx, posting); err != nil {
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
			if err := h.sectorAmounts.PostingPurged(tx, posting); err != nil {
				return fmt.Errorf("purge posting: %w", err)
			}
		}

		return nil
	})
}

func (h harness) amountOfPostingsPerDHTRingSector(
	t *testing.T,
) map[yacymodel.DHTRingSector]int {
	t.Helper()

	var amountPerSector map[yacymodel.DHTRingSector]int
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		amounts, err := h.sectorAmounts.AmountOfPostingsPerDHTRingSector(tx)
		amountPerSector = amounts

		return err
	}); err != nil {
		t.Fatalf("AmountOfPostingsPerDHTRingSector: %v", err)
	}

	return amountPerSector
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

func TestANodeHoldingNoPostingCountsNoSector(t *testing.T) {
	h := openHarness(t)

	if amounts := h.amountOfPostingsPerDHTRingSector(t); len(amounts) != 0 {
		t.Fatalf("amounts = %v, want none", amounts)
	}
}

func TestEachStoredPostingRaisesTheAmountOfItsSector(t *testing.T) {
	h := openHarness(t)
	together := []yacymodel.RWIPosting{postingOf("w1", "u1"), postingOf("w2", "u1")}
	apart := postingOf("w1", "u2")

	h.store(t, together[0], together[1], apart)

	want := map[yacymodel.DHTRingSector]int{
		sectorOf(t, together[0]): 2,
		sectorOf(t, apart):       1,
	}
	if got := h.amountOfPostingsPerDHTRingSector(t); !maps.Equal(got, want) {
		t.Fatalf("amounts = %v, want %v", got, want)
	}
}

func TestEachPurgedPostingLowersTheAmountOfItsSector(t *testing.T) {
	h := openHarness(t)
	remaining := postingOf("w1", "u1")
	h.store(t, remaining, postingOf("w2", "u1"))

	h.purge(t, postingOf("w2", "u1"))

	want := map[yacymodel.DHTRingSector]int{sectorOf(t, remaining): 1}
	if got := h.amountOfPostingsPerDHTRingSector(t); !maps.Equal(got, want) {
		t.Fatalf("amounts = %v, want %v", got, want)
	}
}

func TestTheLastPurgedPostingLeavesItsSectorUncounted(t *testing.T) {
	h := openHarness(t)
	h.store(t, postingOf("w1", "u1"))

	h.purge(t, postingOf("w1", "u1"))

	if amounts := h.amountOfPostingsPerDHTRingSector(t); len(amounts) != 0 {
		t.Fatalf("amounts = %v, want none", amounts)
	}
}

func TestAPostingPurgedAndStoredAgainLeavesTheAmountAlone(t *testing.T) {
	h := openHarness(t)
	posting := postingOf("w1", "u1")
	h.store(t, posting)

	h.purge(t, posting)
	h.store(t, posting)

	want := map[yacymodel.DHTRingSector]int{sectorOf(t, posting): 1}
	if got := h.amountOfPostingsPerDHTRingSector(t); !maps.Equal(got, want) {
		t.Fatalf("amounts = %v, want %v", got, want)
	}
}
