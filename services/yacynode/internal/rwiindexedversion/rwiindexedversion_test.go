package rwiindexedversion_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiindexedversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

func openIndexedVersions(t *testing.T) (*vault.Vault, *rwiindexedversion.IndexedVersions) {
	t.Helper()

	v, err := vault.New(vaultenginetest.EngineRepeatingWrites(memoryvault.OpenEngine(0)), nil)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	versions, err := rwiindexedversion.Open(v)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	return v, versions
}

func urlHash(t *testing.T) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf("https://example.org/page")
	if err != nil {
		t.Fatalf("URLHashOf: %v", err)
	}

	return hash
}

func sampleVersion(t *testing.T) rwiindexedversion.IndexedVersion {
	t.Helper()

	german, err := yacymodel.ParseLanguage("de")
	if err != nil {
		t.Fatalf("ParseLanguage: %v", err)
	}

	return rwiindexedversion.IndexedVersion{
		Fingerprint: pagefingerprint.Fingerprint{
			Modified:      20000,
			LocalLinks:    44,
			ExternalLinks: 3,
			TitleWords:    5,
			Language:      german,
			DocumentType:  yacymodel.DocumentTypeText,
		},
		Origin: rwiorigin.Of("203.0.113.7"),
	}
}

func TestRecordedVersionReadsBackAndCounts(t *testing.T) {
	v, versions := openIndexedVersions(t)
	url := urlHash(t)
	recorded := sampleVersion(t)

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return versions.Record(tx, url, recorded)
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		read, found, err := versions.Of(tx, url)
		if err != nil {
			return err
		}
		if !found || read != recorded {
			t.Errorf("Of = %+v, %v, want %+v", read, found, recorded)
		}
		count, err := versions.Count(tx)
		if count != 1 {
			t.Errorf("Count = %d, want 1", count)
		}

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func TestPurgedURLForgetsItsVersion(t *testing.T) {
	v, versions := openIndexedVersions(t)
	url := urlHash(t)

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		if err := versions.Record(tx, url, sampleVersion(t)); err != nil {
			return err
		}

		return versions.URLPurged(tx, url)
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		_, found, err := versions.Of(tx, url)
		if found {
			t.Error("a purged url kept its indexed version")
		}

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}
