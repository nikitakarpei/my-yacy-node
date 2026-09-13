package rwiimpactorder_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiimpactorder"
)

type harness struct {
	vault       *vault.Vault
	impactOrder rwiimpactorder.ImpactOrderProjection
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

	impactOrder, err := rwiimpactorder.Open(v)
	if err != nil {
		t.Fatalf("rwiimpactorder.Open: %v", err)
	}

	return harness{vault: v, impactOrder: impactOrder}
}

func (h harness) store(t *testing.T, postings ...yacymodel.RWIPosting) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		for _, posting := range postings {
			if err := h.impactOrder.PostingStored(tx, posting); err != nil {
				return fmt.Errorf("store posting: %w", err)
			}
		}

		return nil
	})
}

func (h harness) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (h harness) documentsInImpactOrder(t *testing.T, word yacymodel.Hash) []string {
	t.Helper()

	var documents []string
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		return h.impactOrder.ScanPostingsInImpactOrder(
			tx,
			word,
			func(document yacymodel.URLHash, _ rwiimpactorder.Impact) (bool, error) {
				documents = append(documents, document.String())

				return true, nil
			},
		)
	}); err != nil {
		t.Fatalf("ScanPostingsInImpactOrder: %v", err)
	}

	return documents
}

func (h harness) largestImpactOf(t *testing.T, word yacymodel.Hash) (rwiimpactorder.Impact, bool) {
	t.Helper()

	var (
		largestImpact rwiimpactorder.Impact
		found         bool
	)
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		impact, held, err := h.impactOrder.LargestImpactOf(tx, word)
		largestImpact, found = impact, held

		return err
	}); err != nil {
		t.Fatalf("LargestImpactOf: %v", err)
	}

	return largestImpact, found
}

func documentHashOf(seed string) yacymodel.URLHash {
	address, err := url.Parse("http://example.com/" + seed)
	if err != nil {
		panic(err)
	}

	return yacymodel.URLNormalformOf(address).Hash()
}

func postingOf(word, document string, hits int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash: yacymodel.WordHash(word),
		URLHash:  documentHashOf(document),
		Language: yacymodel.LanguageOfUndeclaredDocument,
		Hits:     hits,
	}
}

func titlePostingOf(word, document string) yacymodel.RWIPosting {
	posting := postingOf(word, document, 1)
	posting.Appearance.AppearsInTitle = true

	return posting
}

func TestPostingsComeBackWithTheMostHitsFirst(t *testing.T) {
	h := openHarness(t)

	h.store(t,
		postingOf("w1", "u1", 1),
		postingOf("w1", "u2", 9),
		postingOf("w1", "u3", 4),
	)

	documents := h.documentsInImpactOrder(t, yacymodel.WordHash("w1"))
	want := []string{
		documentHashOf("u2").String(),
		documentHashOf("u3").String(),
		documentHashOf("u1").String(),
	}
	for position, document := range want {
		if documents[position] != document {
			t.Fatalf("documents = %v, want %v", documents, want)
		}
	}
}

func TestAPostingOfATitleOutweighsAnyAmountOfHits(t *testing.T) {
	h := openHarness(t)

	h.store(t,
		postingOf("w1", "u1", 1000),
		titlePostingOf("w1", "u2"),
	)

	documents := h.documentsInImpactOrder(t, yacymodel.WordHash("w1"))
	if documents[0] != documentHashOf("u2").String() {
		t.Fatalf("documents = %v, want the title posting first", documents)
	}
}

func TestOnlyThePostingsOfTheWordComeBack(t *testing.T) {
	h := openHarness(t)

	h.store(t,
		postingOf("w1", "u1", 1),
		postingOf("w2", "u2", 1),
	)

	if documents := h.documentsInImpactOrder(t, yacymodel.WordHash("w1")); len(documents) != 1 {
		t.Fatalf("documents = %v, want only the posting of w1", documents)
	}
}

func TestAPurgedPostingLeavesTheOrder(t *testing.T) {
	h := openHarness(t)

	h.store(t, postingOf("w1", "u1", 1), postingOf("w1", "u2", 9))
	h.write(t, func(tx *vault.Txn) error {
		return h.impactOrder.PostingPurged(tx, postingOf("w1", "u2", 9))
	})

	documents := h.documentsInImpactOrder(t, yacymodel.WordHash("w1"))
	if len(documents) != 1 || documents[0] != documentHashOf("u1").String() {
		t.Fatalf("documents = %v, want only u1", documents)
	}
}

func TestAPostingPurgedAndStoredWithMoreHitsMovesToItsNewPlace(t *testing.T) {
	h := openHarness(t)

	h.store(t, postingOf("w1", "u1", 9), postingOf("w1", "u2", 1))
	h.write(t, func(tx *vault.Txn) error {
		if err := h.impactOrder.PostingPurged(tx, postingOf("w1", "u2", 1)); err != nil {
			return err
		}

		return h.impactOrder.PostingStored(tx, postingOf("w1", "u2", 99))
	})

	documents := h.documentsInImpactOrder(t, yacymodel.WordHash("w1"))
	if len(documents) != 2 || documents[0] != documentHashOf("u2").String() {
		t.Fatalf("documents = %v, want u2 first and each posting once", documents)
	}
}

func TestTheLargestImpactIsTheImpactOfTheFirstPosting(t *testing.T) {
	h := openHarness(t)

	h.store(t, postingOf("w1", "u1", 1), titlePostingOf("w1", "u2"))

	largestImpact, found := h.largestImpactOf(t, yacymodel.WordHash("w1"))
	if !found {
		t.Fatal("LargestImpactOf found nothing, want the title posting")
	}
	if largestImpact != rwiimpactorder.ImpactOf(titlePostingOf("w1", "u2")) {
		t.Errorf("largest impact = %d, want the impact of the title posting", largestImpact)
	}
}

func TestAWordWithoutPostingsHasNoLargestImpact(t *testing.T) {
	h := openHarness(t)

	if _, found := h.largestImpactOf(t, yacymodel.WordHash("w1")); found {
		t.Fatal("LargestImpactOf found an impact, want none")
	}
}
