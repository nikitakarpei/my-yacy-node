package urlreferences_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlreferences"
)

type harness struct {
	vault      *vault.Vault
	references urlreferences.ReferenceQuery
	admitter   rwipostings.PostingAdmitter
	purger     rwipostings.PostingPurger
}

func openReferences(t *testing.T) harness {
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

	references, projection, err := urlreferences.Open(v)
	if err != nil {
		t.Fatalf("urlreferences.Open: %v", err)
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

	return harness{vault: v, references: references, admitter: admitter, purger: purger}
}

func (h harness) store(t *testing.T, word yacymodel.Hash, url yacymodel.URLHash) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.admitter.Admit(tx, yacymodel.RWIPosting{
			WordHash: word,
			URLHash:  url,
			Language: yacymodel.LanguageOfUndeclaredDocument,
		})
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
		if err != nil {
			return fmt.Errorf("words referencing: %w", err)
		}
		words = found

		return nil
	}); err != nil {
		t.Fatalf("WordsReferencing: %v", err)
	}

	return words
}

func TestWordsReferencingListsStoredWords(t *testing.T) {
	h := openReferences(t)
	url := urlHash("u1")
	h.store(t, yacymodel.WordHash("w1"), url)
	h.store(t, yacymodel.WordHash("w2"), url)
	h.store(t, yacymodel.WordHash("w3"), urlHash("other"))

	if got := h.wordsReferencing(t, url); len(got) != 2 {
		t.Fatalf("words referencing %q = %v, want 2", url, got)
	}
}

func TestPurgedPostingDropsOnlyItsWordFromTheURL(t *testing.T) {
	h := openReferences(t)
	url := urlHash("u1")
	h.store(t, yacymodel.WordHash("w1"), url)
	h.store(t, yacymodel.WordHash("w2"), url)

	h.purge(t, yacymodel.WordHash("w1"), url)

	got := h.wordsReferencing(t, url)
	if len(got) != 1 || got[0] != yacymodel.WordHash("w2") {
		t.Fatalf("words referencing %q after purge = %v, want only w2", url, got)
	}
}

func TestPurgedUnknownPostingIsHarmless(t *testing.T) {
	h := openReferences(t)
	url := urlHash("u1")
	h.store(t, yacymodel.WordHash("w1"), url)

	h.purge(t, yacymodel.WordHash("w2"), url)

	if got := h.wordsReferencing(t, url); len(got) != 1 {
		t.Fatalf("words referencing %q = %v, want w1 kept", url, got)
	}
}

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func TestAPostingPurgedAndStoredAgainLeavesTheReferenceIntact(t *testing.T) {
	h := openReferences(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	h.store(t, word, url)

	h.purge(t, word, url)
	h.store(t, word, url)

	words := h.wordsReferencing(t, url)
	if len(words) != 1 || words[0] != word {
		t.Fatalf("words referencing url = %v, want [%v]", words, word)
	}
}
