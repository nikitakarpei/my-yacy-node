// Package rwiindexedversion records, for each URL indexed from peer postings,
// the fingerprint of the page version whose postings the index holds and the
// origin that sent that version. A purge of the URL forgets the record.
package rwiindexedversion

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

type IndexedVersion struct {
	Fingerprint pagefingerprint.Fingerprint
	Origin      rwiorigin.Origin
}

type IndexedVersions struct {
	versions *vault.Collection[yacymodel.URLHash, IndexedVersion]
}

func Open(v *vault.Vault) (*IndexedVersions, error) {
	versions, err := registerIndexedVersions(v)
	if err != nil {
		return nil, err
	}

	return &IndexedVersions{versions: versions}, nil
}

func (i *IndexedVersions) Of(
	tx *vault.Txn,
	url yacymodel.URLHash,
) (IndexedVersion, bool, error) {
	version, found, err := i.versions.Get(tx, url)
	if err != nil {
		return IndexedVersion{}, false, fmt.Errorf("read indexed version: %w", err)
	}

	return version, found, nil
}

func (i *IndexedVersions) Record(
	tx *vault.Txn,
	url yacymodel.URLHash,
	version IndexedVersion,
) error {
	if _, err := i.versions.Put(tx, url, version); err != nil {
		return fmt.Errorf("record indexed version: %w", err)
	}

	return nil
}

func (i *IndexedVersions) Count(tx *vault.Txn) (int, error) {
	count, err := i.versions.Len(tx)
	if err != nil {
		return 0, fmt.Errorf("count indexed versions: %w", err)
	}

	return count, nil
}

func (i *IndexedVersions) URLStored(
	*vault.Txn,
	yacymodel.URLHash,
	pagefingerprint.Fingerprint,
) error {
	return nil
}

func (i *IndexedVersions) URLPurged(tx *vault.Txn, url yacymodel.URLHash) error {
	if _, err := i.versions.Delete(tx, url); err != nil {
		return fmt.Errorf("forget indexed version: %w", err)
	}

	return nil
}
