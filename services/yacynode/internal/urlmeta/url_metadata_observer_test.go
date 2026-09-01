package urlmeta_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type recordingObserver struct {
	stored []yacymodel.URLHash
	purged []yacymodel.URLHash
	fail   bool
}

func (r *recordingObserver) URLStored(
	tx *vault.Txn,
	hash yacymodel.URLHash,
	_ yacymodel.Optional[yacymodel.CalendarDay],
) error {
	tx.RunAfterCommit(func() { r.stored = append(r.stored, hash) })
	if r.fail {
		return fmt.Errorf("observer refused store")
	}

	return nil
}

func (r *recordingObserver) URLPurged(tx *vault.Txn, hash yacymodel.URLHash) error {
	tx.RunAfterCommit(func() { r.purged = append(r.purged, hash) })
	if r.fail {
		return fmt.Errorf("observer refused purge")
	}

	return nil
}

func openObservedModule(
	t *testing.T,
	watchers ...urlmeta.URLMetadataObserver,
) (*vault.Vault, urlPorts) {
	t.Helper()

	v, err := vault.New(
		vaultenginetest.EngineRepeatingWrites(memoryvault.OpenEngine(0)),
		nil,
	)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	directory, evictor, receiver, err := urlmeta.Open(v, watchers...)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	return v, urlPorts{Directory: directory, Evictor: evictor, Receiver: receiver}
}

func storedURLCount(t *testing.T, v *vault.Vault, directory urlmeta.URLDirectory) int {
	t.Helper()

	var count int
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		stored, err := directory.Count(tx)
		count = stored

		return err
	}); err != nil {
		t.Fatalf("Count: %v", err)
	}

	return count
}

func metadataByHash(
	t *testing.T,
	v *vault.Vault,
	directory urlmeta.URLDirectory,
	hashes []yacymodel.URLHash,
) []yacymodel.URLMetadata {
	t.Helper()

	var rows []yacymodel.URLMetadata
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		stored, err := directory.MetadataByHash(tx, hashes)
		rows = stored

		return err
	}); err != nil {
		t.Fatalf("MetadataByHash: %v", err)
	}

	return rows
}
