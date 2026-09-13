package rwipostings_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type harness struct {
	vault    *vault.Vault
	index    rwipostings.PostingIndex
	admitter rwipostings.PostingAdmitter
	purger   rwipostings.PostingPurger
	observer *recordingObserver
}

func (h harness) rwiCount(t *testing.T) int {
	t.Helper()

	var count int
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		measured, err := h.index.RWICount(tx)
		count = measured

		return err
	}); err != nil {
		t.Fatalf("RWICount: %v", err)
	}

	return count
}

func (h harness) postingOf(
	t *testing.T,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool) {
	t.Helper()

	var (
		entry yacymodel.RWIPosting
		found bool
	)
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		stored, ok, err := h.index.PostingOf(tx, word, url)
		entry, found = stored, ok

		return err
	}); err != nil {
		t.Fatalf("PostingOf: %v", err)
	}

	return entry, found
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

	observer := &recordingObserver{}
	index, admitter, purger, err := rwipostings.Open(v, observer)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}

	return harness{vault: v, index: index, admitter: admitter, purger: purger, observer: observer}
}

type recordingObserver struct {
	stored  []yacymodel.RWIPosting
	updated []postingUpdate
	purged  []yacymodel.RWIPosting
}

type postingUpdate struct {
	previous yacymodel.RWIPosting
	current  yacymodel.RWIPosting
}

func (o *recordingObserver) PostingStored(_ *vault.Txn, posting yacymodel.RWIPosting) error {
	o.stored = append(o.stored, posting)

	return nil
}

func (o *recordingObserver) PostingUpdated(
	_ *vault.Txn,
	previous yacymodel.RWIPosting,
	current yacymodel.RWIPosting,
) error {
	o.updated = append(o.updated, postingUpdate{previous: previous, current: current})

	return nil
}

func (o *recordingObserver) PostingPurged(_ *vault.Txn, posting yacymodel.RWIPosting) error {
	o.purged = append(o.purged, posting)

	return nil
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
	address, err := url.Parse("http://example.com/" + seed)
	if err != nil {
		panic(err)
	}

	return yacymodel.URLNormalformOf(address).Hash()
}

func posting(word, urlSeed string) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:   yacymodel.WordHash(word),
		URLHash:    urlHash(urlSeed),
		Language:   yacymodel.LanguageOfUndeclaredDocument,
		LocalLinks: 1,
		Hits:       1,
	}
}
