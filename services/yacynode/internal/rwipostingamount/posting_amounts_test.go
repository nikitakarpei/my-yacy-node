package rwipostingamount_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingamount"
)

type harness struct {
	vault          *vault.Vault
	postingAmounts rwipostingamount.PostingAmountProjection
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

	postingAmounts, err := rwipostingamount.Open(v)
	if err != nil {
		t.Fatalf("rwipostingamount.Open: %v", err)
	}

	return harness{vault: v, postingAmounts: postingAmounts}
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
			if err := h.postingAmounts.PostingStored(tx, posting); err != nil {
				return fmt.Errorf("store posting: %w", err)
			}
		}

		return nil
	})
}

func (h harness) amountOfPostingsOf(t *testing.T, word yacymodel.Hash) int {
	t.Helper()

	var amountOfPostings int
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		amount, err := h.postingAmounts.AmountOfPostingsOf(tx, word)
		amountOfPostings = amount

		return err
	}); err != nil {
		t.Fatalf("AmountOfPostingsOf: %v", err)
	}

	return amountOfPostings
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

func TestAWordTheNodeHoldsNoPostingOfCountsNone(t *testing.T) {
	h := openHarness(t)

	if amount := h.amountOfPostingsOf(t, yacymodel.WordHash("w1")); amount != 0 {
		t.Fatalf("amount = %d, want 0", amount)
	}
}

func TestEachStoredPostingRaisesTheAmountOfItsWord(t *testing.T) {
	h := openHarness(t)

	h.store(t, postingOf("w1", "u1"), postingOf("w1", "u2"), postingOf("w2", "u1"))

	if amount := h.amountOfPostingsOf(t, yacymodel.WordHash("w1")); amount != 2 {
		t.Errorf("amount of w1 = %d, want 2", amount)
	}
	if amount := h.amountOfPostingsOf(t, yacymodel.WordHash("w2")); amount != 1 {
		t.Errorf("amount of w2 = %d, want 1", amount)
	}
}

func TestEachPurgedPostingLowersTheAmountOfItsWord(t *testing.T) {
	h := openHarness(t)

	h.store(t, postingOf("w1", "u1"), postingOf("w1", "u2"))
	h.write(t, func(tx *vault.Txn) error {
		return h.postingAmounts.PostingPurged(tx, postingOf("w1", "u2"))
	})

	if amount := h.amountOfPostingsOf(t, yacymodel.WordHash("w1")); amount != 1 {
		t.Errorf("amount of w1 = %d, want 1", amount)
	}
}

func TestTheLastPurgedPostingLeavesTheWordAtNone(t *testing.T) {
	h := openHarness(t)

	h.store(t, postingOf("w1", "u1"))
	h.write(t, func(tx *vault.Txn) error {
		return h.postingAmounts.PostingPurged(tx, postingOf("w1", "u1"))
	})

	if amount := h.amountOfPostingsOf(t, yacymodel.WordHash("w1")); amount != 0 {
		t.Errorf("amount of w1 = %d, want 0", amount)
	}
}

func TestAnUpdatedPostingLeavesTheAmountAlone(t *testing.T) {
	h := openHarness(t)

	h.store(t, postingOf("w1", "u1"))
	h.write(t, func(tx *vault.Txn) error {
		return h.postingAmounts.PostingUpdated(
			tx,
			postingOf("w1", "u1"),
			postingOf("w1", "u1"),
		)
	})

	if amount := h.amountOfPostingsOf(t, yacymodel.WordHash("w1")); amount != 1 {
		t.Errorf("amount of w1 = %d, want 1", amount)
	}
}
