package urlmeta

import (
	"context"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

func TestPurgeNotifiesObserverOfDeletedURLs(t *testing.T) {
	ctx := context.Background()
	observer := &recordingObserver{}
	v, module := openObservedModule(t, observer)
	row := urlMetadata("a")
	if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{row}); err != nil {
		t.Fatalf("Intake: %v", err)
	}

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		if _, purgeErr := module.Evictor.Purge(
			ctx,
			tx,
			[]yacymodel.Hash{metadataHash(t, row)},
		); purgeErr != nil {
			return fmt.Errorf("purge: %w", purgeErr)
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(observer.purged) != 1 || observer.purged[0] != metadataHash(t, row) {
		t.Fatalf("purged = %v, want one matching hash", observer.purged)
	}
}

func TestPurgeSurvivesObserverFailure(t *testing.T) {
	ctx := context.Background()
	observer := &recordingObserver{fail: true}
	v, module := openObservedModule(t, observer)
	row := urlMetadata("a")
	if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{row}); err != nil {
		t.Fatalf("Intake: %v", err)
	}

	var result PurgeResult
	if err := v.Update(ctx, func(tx *vault.Txn) error {
		purged, purgeErr := module.Evictor.Purge(ctx, tx, []yacymodel.Hash{metadataHash(t, row)})
		result = purged
		if purgeErr != nil {
			return fmt.Errorf("purge: %w", purgeErr)
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result.URLsDeleted != 1 {
		t.Fatalf("URLsDeleted = %d, want 1 despite observer failure", result.URLsDeleted)
	}
}

func TestPurgeDeletesRows(t *testing.T) {
	ctx := context.Background()
	v, module := openObservedModule(t)
	row := urlMetadata("a")
	if _, err := module.Receiver.Receive(
		ctx,
		[]yacymodel.URLMetadata{row, urlMetadata("b")},
	); err != nil {
		t.Fatalf("Intake: %v", err)
	}

	var result PurgeResult
	if err := v.Update(ctx, func(tx *vault.Txn) error {
		purged, purgeErr := module.Evictor.Purge(ctx, tx, []yacymodel.Hash{metadataHash(t, row)})
		result = purged
		if purgeErr != nil {
			return fmt.Errorf("purge: %w", purgeErr)
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result.URLsDeleted != 1 {
		t.Fatalf("URLsDeleted = %d, want 1", result.URLsDeleted)
	}

	count, err := module.Directory.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("Count = %d, want 1", count)
	}
}
