package rwipostings_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func (h harness) writeAtOnce(t *testing.T, postings ...yacymodel.RWIPosting) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.writer.Write(tx, postings...)
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func (h harness) purge(t *testing.T, word yacymodel.Hash, url yacymodel.URLHash) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		_, err := h.purger.PurgePosting(tx, word, url)

		return err
	}); err != nil {
		t.Fatalf("PurgePosting: %v", err)
	}
}

func (h harness) purgePostingsOf(t *testing.T, url yacymodel.URLHash) int {
	t.Helper()

	var amountOfPurgedPostings int
	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		purged, err := h.purger.PurgePostingsOf(tx, url)
		amountOfPurgedPostings = purged

		return err
	}); err != nil {
		t.Fatalf("PurgePostingsOf: %v", err)
	}

	return amountOfPurgedPostings
}

func numberedWordsOf(amount int) []yacymodel.Hash {
	words := make([]yacymodel.Hash, 0, amount)
	for number := range amount {
		words = append(words, yacymodel.WordHash(fmt.Sprintf("w%02d", number)))
	}

	return words
}

func postingsOfWordsOn(words []yacymodel.Hash, url yacymodel.URLHash) []yacymodel.RWIPosting {
	postings := make([]yacymodel.RWIPosting, 0, len(words))
	for _, word := range words {
		postings = append(postings, postingOfWordOn(word, url))
	}

	return postings
}

func TestPurgingThePostingsOfAURLLeavesOtherURLs(t *testing.T) {
	h := openHarness(t)
	url, other := urlHash("u1"), urlHash("other")
	words := numberedWordsOf(3)
	h.writeEach(t, postingsOfWordsOn(words, url)...)
	h.writeEach(t, postingOfWordOn(words[0], other))

	if purged := h.purgePostingsOf(t, url); purged != len(words) {
		t.Fatalf("purged %d postings of %q, want %d", purged, url, len(words))
	}
	for _, word := range words {
		if _, found := h.postingOf(t, word, url); found {
			t.Fatalf("posting of %v on %q outlived the purge", word, url)
		}
	}
	if _, found := h.postingOf(t, words[0], other); !found {
		t.Fatalf("posting on %q was purged with %q", other, url)
	}
}

func TestAPurgedPostingLeavesTheOtherPostingsOfItsURL(t *testing.T) {
	h := openHarness(t)
	url := urlHash("u1")
	h.writeEach(t, postingOfWordOn(yacymodel.WordHash("w1"), url))
	h.writeEach(t, postingOfWordOn(yacymodel.WordHash("w2"), url))

	h.purge(t, yacymodel.WordHash("w1"), url)

	if purged := h.purgePostingsOf(t, url); purged != 1 {
		t.Fatalf("purged %d postings of %q after one was purged, want only w2", purged, url)
	}
}

func TestAPageWrittenAtOnceHoldsEachPostingOnce(t *testing.T) {
	h := openHarness(t)
	url := urlHash("u1")
	words := numberedWordsOf(12)
	h.writeEach(t, postingsOfWordsOn(words[:3], url)...)

	h.writeAtOnce(t, append(postingsOfWordsOn(words, url), postingOfWordOn(words[5], url))...)

	for _, word := range words {
		if _, found := h.postingOf(t, word, url); !found {
			t.Fatalf("posting of %v on url not found", word)
		}
	}
	if purged := h.purgePostingsOf(t, url); purged != len(words) {
		t.Fatalf(
			"purged %d postings of the url, want each of the %d words once",
			purged,
			len(words),
		)
	}
}
