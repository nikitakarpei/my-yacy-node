package rwipostings_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func (h harness) admitAtOnce(t *testing.T, postings ...yacymodel.RWIPosting) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.admitter.Admit(tx, postings...)
	}); err != nil {
		t.Fatalf("Admit: %v", err)
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

func (h harness) wordsReferencing(t *testing.T, url yacymodel.URLHash) []yacymodel.Hash {
	t.Helper()

	var words []yacymodel.Hash
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		found, err := h.references.WordsReferencing(tx, url)
		words = found

		return err
	}); err != nil {
		t.Fatalf("WordsReferencing: %v", err)
	}

	return words
}

func numberedWordsOf(amount int) []yacymodel.Hash {
	words := make([]yacymodel.Hash, 0, amount)
	for number := range amount {
		words = append(words, yacymodel.WordHash(fmt.Sprintf("w%02d", number)))
	}

	return words
}

func sortedWords(words []yacymodel.Hash) []yacymodel.Hash {
	return slices.SortedFunc(slices.Values(words), func(a, b yacymodel.Hash) int {
		return strings.Compare(a.String(), b.String())
	})
}

func postingsOfWordsOn(words []yacymodel.Hash, url yacymodel.URLHash) []yacymodel.RWIPosting {
	postings := make([]yacymodel.RWIPosting, 0, len(words))
	for _, word := range words {
		postings = append(postings, postingOfWordOn(word, url))
	}

	return postings
}

func TestWordsReferencingListsTheWordsOfOneURL(t *testing.T) {
	h := openHarness(t)
	url := urlHash("u1")
	h.admit(t, postingOfWordOn(yacymodel.WordHash("w1"), url))
	h.admit(t, postingOfWordOn(yacymodel.WordHash("w2"), url))
	h.admit(t, postingOfWordOn(yacymodel.WordHash("w3"), urlHash("other")))

	if got := h.wordsReferencing(t, url); len(got) != 2 {
		t.Fatalf("words referencing %q = %v, want 2", url, got)
	}
}

func TestAPurgedPostingDropsOnlyItsWordFromTheURL(t *testing.T) {
	h := openHarness(t)
	url := urlHash("u1")
	h.admit(t, postingOfWordOn(yacymodel.WordHash("w1"), url))
	h.admit(t, postingOfWordOn(yacymodel.WordHash("w2"), url))

	h.purge(t, yacymodel.WordHash("w1"), url)

	got := h.wordsReferencing(t, url)
	if len(got) != 1 || got[0] != yacymodel.WordHash("w2") {
		t.Fatalf("words referencing %q after purge = %v, want only w2", url, got)
	}
}

func TestAPageAdmittedAtOnceHoldsEachPostingOnce(t *testing.T) {
	h := openHarness(t)
	url := urlHash("u1")
	words := numberedWordsOf(12)
	h.admit(t, postingsOfWordsOn(words[:3], url)...)

	h.admitAtOnce(t, append(postingsOfWordsOn(words, url), postingOfWordOn(words[5], url))...)

	for _, word := range words {
		if _, found := h.postingOf(t, word, url); !found {
			t.Fatalf("posting of %v on url not found", word)
		}
	}
	if got := sortedWords(h.wordsReferencing(t, url)); !slices.Equal(got, sortedWords(words)) {
		t.Fatalf("words referencing url = %v, want %v", got, words)
	}
}
