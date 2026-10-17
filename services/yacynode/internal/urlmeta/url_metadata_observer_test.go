package urlmeta_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type recordingObserver struct {
	stored         []yacymodel.URLHash
	storedVersions []urlmeta.CurrentVersion
	purged         []yacymodel.URLHash
	events         []string
	fail           bool
}

func (r *recordingObserver) URLStored(
	tx *vault.Txn,
	hash yacymodel.URLHash,
	version urlmeta.CurrentVersion,
) error {
	tx.RunAfterCommit(func() {
		r.stored = append(r.stored, hash)
		r.storedVersions = append(r.storedVersions, version)
		r.events = append(r.events, "stored")
	})
	if r.fail {
		return fmt.Errorf("observer refused store")
	}

	return nil
}

func (r *recordingObserver) URLPurged(tx *vault.Txn, hash yacymodel.URLHash) error {
	tx.RunAfterCommit(func() {
		r.purged = append(r.purged, hash)
		r.events = append(r.events, "purged")
	})
	if r.fail {
		return fmt.Errorf("observer refused purge")
	}

	return nil
}

type recordedOutcomes map[urlmeta.Outcome]int

func (r recordedOutcomes) ObserveOutcome(outcome urlmeta.Outcome, records int) {
	r[outcome] += records
}

var (
	sendingNetwork = rwiorigin.Of("203.0.113.7")
	otherNetwork   = rwiorigin.Of("198.51.100.9")
)

func openObservedModule(
	t *testing.T,
	watchers ...urlmeta.URLMetadataObserver,
) (*vault.Vault, urlPorts) {
	t.Helper()

	return openModuleAt(t, intakeClock, watchers...)
}

func openModuleAt(
	t *testing.T,
	now func() time.Time,
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

	outcomes := recordedOutcomes{}
	directory, evictor, admitter, receiver, err := urlmeta.Open(
		context.Background(),
		v,
		now,
		urlmeta.SwapLimit{FloorPerHour: 1},
		outcomes,
		nil,
		watchers...)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	return v, urlPorts{
		Directory: directory,
		Evictor:   evictor,
		Admitter:  admitter,
		Receiver:  receiver,
		Outcomes:  outcomes,
	}
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

func metadataPerHash(
	t *testing.T,
	v *vault.Vault,
	directory urlmeta.URLDirectory,
	hashes []yacymodel.URLHash,
) map[yacymodel.URLHash]yacymodel.URLMetadata {
	t.Helper()

	var rows map[yacymodel.URLHash]yacymodel.URLMetadata
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		stored, err := directory.MetadataPerHash(tx, hashes)
		rows = stored

		return err
	}); err != nil {
		t.Fatalf("MetadataPerHash: %v", err)
	}

	return rows
}
