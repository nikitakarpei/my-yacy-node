package rwipostings_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type recordingObserver struct {
	purged []yacymodel.Hash
}

func (o *recordingObserver) PostingStored(
	_ *vault.Txn,
	_ yacymodel.Hash,
	_ yacymodel.URLHash,
) error {
	return nil
}

func (o *recordingObserver) PostingPurged(
	_ *vault.Txn,
	word yacymodel.Hash,
	_ yacymodel.URLHash,
) error {
	o.purged = append(o.purged, word)

	return nil
}

func TestPurgePostingDropsPostingAndNotifies(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t)

	h.admit(t,
		posting("w1", "u1"),
		posting("w1", "u2"),
		posting("w2", "u1"),
	)

	word := yacymodel.WordHash("w1")
	url := urlHash("u1")
	var deleted bool
	if err := h.vault.Update(ctx, func(tx *vault.Txn) error {
		dropped, err := h.purger.PurgePosting(tx, word, url)
		if err != nil {
			return fmt.Errorf("purge posting: %w", err)
		}
		deleted = dropped

		return nil
	}); err != nil {
		t.Fatalf("PurgePosting: %v", err)
	}
	if !deleted {
		t.Fatal("PurgePosting reported nothing deleted, want the posting dropped")
	}

	rwiCount, err := h.index.RWICount(ctx)
	if err != nil {
		t.Fatalf("RWICount: %v", err)
	}
	if rwiCount != 2 {
		t.Fatalf("RWICount = %d, want 2", rwiCount)
	}
	if len(h.observer.purged) != 1 || h.observer.purged[0] != word {
		t.Fatalf("purged observers = %v, want one notification for %q", h.observer.purged, word)
	}
}
