package rwipostings

import (
	"context"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type harness struct {
	vault    *vault.Vault
	index    PostingIndex
	admitter PostingAdmitter
	purger   PostingPurger
	observer *recordingObserver
}

func openHarness(t *testing.T) harness {
	t.Helper()

	v, err := memvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	observer := &recordingObserver{}
	index, admitter, purger, err := Open(v, observer)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}

	return harness{vault: v, index: index, admitter: admitter, purger: purger, observer: observer}
}

func (h harness) admit(t *testing.T, postings ...yacymodel.RWIPosting) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		for _, entry := range postings {
			if err := h.admitter.Admit(tx, entry); err != nil {
				return fmt.Errorf("admit posting: %w", err)
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Admit: %v", err)
	}
}

func urlHash(seed string) yacymodel.URLHash {
	hash, err := yacymodel.HashURL("http://example.com/" + seed)
	if err != nil {
		panic(err)
	}

	return hash
}

func posting(word, urlSeed string) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:   yacymodel.WordHash(word),
		URLHash:    urlHash(urlSeed),
		LocalLinks: 1,
		Hits:       1,
	}
}
