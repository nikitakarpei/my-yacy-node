package vault

import (
	"context"
	"fmt"
)

const projectionVersionsBucket Name = "vault_projection_versions"

const rebuildBatchRecords = 1000

var projectionVersionKeys = SingleKey(TextKeyPartFrom(
	func(bucket Name) string { return string(bucket) },
	func(text string) (Name, error) { return Name(text), nil },
)).KeyLayout()

type projectionVersionCodec struct{}

func (projectionVersionCodec) Encode(version string) ([]byte, error) {
	return []byte(version), nil
}

func (projectionVersionCodec) Decode(stored []byte) (string, error) {
	return string(stored), nil
}

func (v *Vault) registerProjectionVersions() (*Collection[Name, string], error) {
	return v.RegisterCollection(
		projectionVersionsBucket,
		projectionVersionKeys,
		projectionVersionCodec{},
	)
}

func (v *Vault) rebuildStaleProjections[K, V any](
	ctx context.Context,
	collection *Collection[K, V],
	projections Projections[K, V],
) error {
	versions, err := v.projectionVersions()
	if err != nil {
		return err
	}
	stale, err := v.staleProjectionsAmong(ctx, versions, projections.buckets())
	if err != nil || len(stale) == 0 {
		return err
	}
	for _, bucket := range stale {
		if err := v.updateInBatches(ctx, bucket.clearFrom); err != nil {
			return fmt.Errorf("clear stale projection: %w", err)
		}
	}
	staleProjections := projections.narrowedTo(stale)
	if err := v.updateInBatches(ctx, func(tx *Txn, firstKey []byte, limit int) ([]byte, error) {
		return collection.projectRecordsFrom(tx, firstKey, limit, staleProjections)
	}); err != nil {
		return fmt.Errorf("rebuild stale projections: %w", err)
	}

	return v.recordProjectionVersions(ctx, versions, stale)
}

func (v *Vault) staleProjectionsAmong(
	ctx context.Context,
	versions *Collection[Name, string],
	buckets []projectionBucket,
) ([]projectionBucket, error) {
	var stale []projectionBucket
	if err := v.View(ctx, func(tx *Txn) error {
		stale = nil
		for _, bucket := range buckets {
			isStale, err := bucket.staleIn(tx, versions)
			if err != nil {
				return err
			}
			if isStale {
				stale = append(stale, bucket)
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("read projection versions: %w", err)
	}

	return stale, nil
}

func (v *Vault) updateInBatches(
	ctx context.Context,
	batch func(tx *Txn, firstKey []byte, limit int) (nextKey []byte, err error),
) error {
	var firstKey []byte
	for {
		var nextKey []byte
		if err := v.Update(ctx, func(tx *Txn) error {
			tx.calledWriteOperation = true
			var err error
			nextKey, err = batch(tx, firstKey, rebuildBatchRecords)

			return err
		}); err != nil {
			return err
		}
		if nextKey == nil {
			return nil
		}
		firstKey = nextKey
	}
}

func (v *Vault) recordProjectionVersions(
	ctx context.Context,
	versions *Collection[Name, string],
	rebuilt []projectionBucket,
) error {
	if err := v.Update(ctx, func(tx *Txn) error {
		for _, bucket := range rebuilt {
			if err := bucket.recordVersionIn(tx, versions); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("record projection versions: %w", err)
	}

	return nil
}
