package urlmeta_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
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
	_ *vault.Txn,
	hash yacymodel.URLHash,
	_ yacymodel.Optional[yacymodel.CalendarDay],
) error {
	r.stored = append(r.stored, hash)
	if r.fail {
		return fmt.Errorf("observer refused store")
	}

	return nil
}

func (r *recordingObserver) URLPurged(_ *vault.Txn, hash yacymodel.URLHash) error {
	r.purged = append(r.purged, hash)
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

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
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
