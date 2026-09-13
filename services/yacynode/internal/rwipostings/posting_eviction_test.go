package rwipostings_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingUpdate struct {
	previous yacymodel.RWIPosting
	current  yacymodel.RWIPosting
}

type recordingObserver struct {
	stored  []yacymodel.RWIPosting
	updated []postingUpdate
	purged  []yacymodel.RWIPosting
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
	var postingWasPurged bool
	if err := h.vault.Update(ctx, func(tx *vault.Txn) error {
		var err error
		postingWasPurged, err = h.purger.PurgePosting(tx, word, url)
		if err != nil {
			return fmt.Errorf("purge posting: %w", err)
		}

		return nil
	}); err != nil {
		t.Fatalf("PurgePosting: %v", err)
	}
	if !postingWasPurged {
		t.Fatal("PurgePosting reported nothing deleted, want the posting dropped")
	}

	if rwiCount := h.rwiCount(t); rwiCount != 2 {
		t.Fatalf("RWICount = %d, want 2", rwiCount)
	}
	if len(h.observer.purged) != 1 {
		t.Fatalf("purged notifications = %d, want 1", len(h.observer.purged))
	}
	if departed := h.observer.purged[0]; departed != posting("w1", "u1") {
		t.Fatalf("purged notification carried %+v, want the posting that was dropped", departed)
	}
}

func TestPurgingAPostingThatIsNotHeldNotifiesNobody(t *testing.T) {
	h := openHarness(t)

	h.admit(t, posting("w1", "u1"))

	var postingWasPurged bool
	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		var err error
		postingWasPurged, err = h.purger.PurgePosting(
			tx, yacymodel.WordHash("w2"), urlHash("u2"),
		)
		if err != nil {
			return fmt.Errorf("purge posting: %w", err)
		}

		return nil
	}); err != nil {
		t.Fatalf("PurgePosting: %v", err)
	}

	if postingWasPurged {
		t.Fatal("PurgePosting reported a deletion, want none")
	}
	if len(h.observer.purged) != 0 {
		t.Fatalf("purged notifications = %v, want none", h.observer.purged)
	}
}
