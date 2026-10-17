package urlmeta

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type urlDirectory struct {
	collection *vault.Collection[yacymodel.URLHash, storedURLMetadata]
	observers  observers
}

func (d urlDirectory) MetadataPerHash(
	tx *vault.Txn,
	hashes []yacymodel.URLHash,
) (map[yacymodel.URLHash]yacymodel.URLMetadata, error) {
	metadata := make(map[yacymodel.URLHash]yacymodel.URLMetadata, len(hashes))
	for _, hash := range hashes {
		stored, ok, err := d.collection.Get(tx, hash)
		if err != nil {
			return nil, fmt.Errorf("read url metadata: %w", err)
		}
		if !ok {
			continue
		}
		stored.metadata.Hash = hash
		metadata[hash] = stored.metadata
	}

	return metadata, nil
}

func (d urlDirectory) CurrentVersionsOf(
	tx *vault.Txn,
	hashes []yacymodel.URLHash,
) (map[yacymodel.URLHash]CurrentVersion, error) {
	versions := make(map[yacymodel.URLHash]CurrentVersion, len(hashes))
	for _, hash := range hashes {
		stored, ok, err := d.collection.Get(tx, hash)
		if err != nil {
			return nil, fmt.Errorf("read url metadata: %w", err)
		}
		if ok {
			versions[hash] = currentVersionOf(stored)
		}
	}

	return versions, nil
}

func (d urlDirectory) Count(tx *vault.Txn) (int, error) {
	count, err := d.collection.Len(tx)
	if err != nil {
		return 0, fmt.Errorf("read url metadata length: %w", err)
	}

	return count, nil
}

func (d urlDirectory) Purge(
	_ context.Context,
	tx *vault.Txn,
	urls []yacymodel.URLHash,
) (PurgeResult, error) {
	var result PurgeResult
	for _, hash := range urls {
		wasDeleted, err := d.collection.Delete(tx, hash)
		if err != nil {
			return PurgeResult{}, fmt.Errorf("delete url metadata: %w", err)
		}
		if !wasDeleted {
			continue
		}
		if err := d.observers.purged(tx, hash); err != nil {
			return PurgeResult{}, err
		}
		result.URLsDeleted++
	}

	return result, nil
}

func (d urlDirectory) Write(
	tx *vault.Txn,
	metadata yacymodel.URLMetadata,
	source Source,
) (CurrentVersion, error) {
	stored := storedURLMetadata{metadata: metadata, source: source}
	if _, err := d.collection.Put(tx, metadata.Hash, stored); err != nil {
		return CurrentVersion{}, fmt.Errorf("write url metadata: %w", err)
	}

	return currentVersionOf(stored), nil
}

var (
	_ URLDirectory      = urlDirectory{}
	_ URLEvictor        = urlDirectory{}
	_ URLMetadataWriter = urlDirectory{}
)
