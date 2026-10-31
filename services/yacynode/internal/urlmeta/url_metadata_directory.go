package urlmeta

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta/urlmetarecord"
)

type urlDirectory struct {
	collection *vault.Collection[yacymodel.URLHash, urlmetarecord.Record]
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
		stored.Metadata.Hash = hash
		metadata[hash] = stored.Metadata
	}

	return metadata, nil
}

func (d urlDirectory) VersionsOf(
	tx *vault.Txn,
	hashes []yacymodel.URLHash,
) (map[yacymodel.URLHash]pageversion.Version, error) {
	versions := make(map[yacymodel.URLHash]pageversion.Version, len(hashes))
	for _, hash := range hashes {
		stored, ok, err := d.collection.Get(tx, hash)
		if err != nil {
			return nil, fmt.Errorf("read url metadata: %w", err)
		}
		if ok {
			versions[hash] = pageversion.FromMetadata(stored.Metadata, stored.Source)
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
	source pageversion.Source,
) error {
	stored := urlmetarecord.Record{Metadata: metadata, Source: source}
	if _, err := d.collection.Put(tx, metadata.Hash, stored); err != nil {
		return fmt.Errorf("write url metadata: %w", err)
	}

	return nil
}

var (
	_ URLDirectory      = urlDirectory{}
	_ URLEvictor        = urlDirectory{}
	_ URLMetadataWriter = urlDirectory{}
)
