package rwipostingamount_test

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingamount"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type harness struct {
	vault          *vault.Vault
	postingAmounts rwipostingamount.PostingAmountQuery
	wordAmounts    rwipostingamount.WordAmountEstimate
	admitter       rwipostings.PostingAdmitter
	purger         rwipostings.PostingPurger
}

func openHarness(t *testing.T) harness {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	return openHarnessOn(t, v)
}

func openHarnessOnDisk(t *testing.T) harness {
	t.Helper()

	engine, err := pebblevault.OpenEngine(t.TempDir(), 0, pebblevault.MachineLimits{}, nil, nil)
	if err != nil {
		t.Fatalf("pebblevault.OpenEngine: %v", err)
	}
	v, err := vault.New(engine, nil)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	return openHarnessOn(t, v)
}

func openHarnessOn(t *testing.T, v *vault.Vault) harness {
	t.Helper()

	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	postingAmounts, wordAmounts, projection, err := rwipostingamount.Open(v)
	if err != nil {
		t.Fatalf("rwipostingamount.Open: %v", err)
	}
	_, admitter, purger, err := rwipostings.Open(
		context.Background(),
		v,
		yacymodel.DHTRingPartitions(1),
		[]vault.Projection[yacymodel.RWIPosting]{projection},
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}

	return harness{
		vault:          v,
		postingAmounts: postingAmounts,
		wordAmounts:    wordAmounts,
		admitter:       admitter,
		purger:         purger,
	}
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
			if err := h.admitter.Admit(tx, posting); err != nil {
				return fmt.Errorf("store posting: %w", err)
			}
		}

		return nil
	})
}

func (h harness) purge(t *testing.T, posting yacymodel.RWIPosting) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		_, err := h.purger.PurgePosting(tx, posting.WordHash, posting.URLHash)

		return err
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

func (h harness) estimatedAmountOfWords(t *testing.T) int {
	t.Helper()

	var amountOfWords int
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		amount, err := h.wordAmounts.EstimatedAmountOfWords(tx)
		amountOfWords = amount

		return err
	}); err != nil {
		t.Fatalf("EstimatedAmountOfWords: %v", err)
	}

	return amountOfWords
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
	h.purge(t, postingOf("w1", "u2"))

	if amount := h.amountOfPostingsOf(t, yacymodel.WordHash("w1")); amount != 1 {
		t.Errorf("amount of w1 = %d, want 1", amount)
	}
}

func TestTheLastPurgedPostingLeavesTheWordAtNone(t *testing.T) {
	h := openHarness(t)

	h.store(t, postingOf("w1", "u1"))
	h.purge(t, postingOf("w1", "u1"))

	if amount := h.amountOfPostingsOf(t, yacymodel.WordHash("w1")); amount != 0 {
		t.Errorf("amount of w1 = %d, want 0", amount)
	}
}

func TestAPostingPurgedAndStoredAgainLeavesTheAmountAlone(t *testing.T) {
	h := openHarness(t)

	h.store(t, postingOf("w1", "u1"))
	h.write(t, func(tx *vault.Txn) error {
		posting := postingOf("w1", "u1")
		if _, err := h.purger.PurgePosting(tx, posting.WordHash, posting.URLHash); err != nil {
			return err
		}

		return h.admitter.Admit(tx, posting)
	})

	if amount := h.amountOfPostingsOf(t, yacymodel.WordHash("w1")); amount != 1 {
		t.Errorf("amount of w1 = %d, want 1", amount)
	}
}

func TestASmallIndexCountsEachWordOnceHoweverManyPostingsItHas(t *testing.T) {
	h := openHarness(t)

	h.store(
		t,
		postingOf("w1", "u1"),
		postingOf("w1", "u2"),
		postingOf("w1", "u3"),
		postingOf("w2", "u1"),
	)

	if amount := h.estimatedAmountOfWords(t); amount != 2 {
		t.Errorf("amount of words = %d, want 2", amount)
	}
}

func TestAWordLeavesTheAmountOfWordsWithItsLastPosting(t *testing.T) {
	h := openHarness(t)
	h.store(t, postingOf("w1", "u1"), postingOf("w1", "u2"), postingOf("w2", "u1"))

	h.purge(t, postingOf("w1", "u1"))
	if amount := h.estimatedAmountOfWords(t); amount != 2 {
		t.Errorf("amount of words after one of two postings left = %d, want 2", amount)
	}
	h.purge(t, postingOf("w1", "u2"))
	if amount := h.estimatedAmountOfWords(t); amount != 1 {
		t.Errorf("amount of words after the last posting of w1 left = %d, want 1", amount)
	}
}

const (
	wordsOfALargeIndex  = 40_000
	estimateToleratedBy = 0.1
)

func TestALargeIndexEstimatesItsWordsClosely(t *testing.T) {
	h := openHarnessOnDisk(t)

	h.store(t, postingsOfWords(wordsOfALargeIndex, func(word yacymodel.Hash) yacymodel.Hash {
		return word
	})...)

	assertEstimateOfLargeIndex(t, h.estimatedAmountOfWords(t))
}

func TestALargeIndexOnOneArcOfTheRingEstimatesItsWordsClosely(t *testing.T) {
	h := openHarnessOnDisk(t)

	h.store(t, postingsOfWords(wordsOfALargeIndex, func(word yacymodel.Hash) yacymodel.Hash {
		onArc, err := yacymodel.ParseHash("Q" + word.String()[1:])
		if err != nil {
			panic(err)
		}

		return onArc
	})...)

	assertEstimateOfLargeIndex(t, h.estimatedAmountOfWords(t))
}

func postingsOfWords(
	amount int,
	placed func(yacymodel.Hash) yacymodel.Hash,
) []yacymodel.RWIPosting {
	postings := make([]yacymodel.RWIPosting, 0, amount)
	for number := range amount {
		posting := postingOf(fmt.Sprintf("word%d", number), "u1")
		posting.WordHash = placed(posting.WordHash)
		postings = append(postings, posting)
	}

	return postings
}

func assertEstimateOfLargeIndex(t *testing.T, estimate int) {
	t.Helper()

	if math.Abs(float64(estimate-wordsOfALargeIndex)) > estimateToleratedBy*wordsOfALargeIndex {
		t.Errorf("estimated amount of words = %d, want %d within %.0f%%",
			estimate, wordsOfALargeIndex, estimateToleratedBy*100)
	}
}
