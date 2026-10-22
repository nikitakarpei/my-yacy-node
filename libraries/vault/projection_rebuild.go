package vault

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const rebuildBatchRecords = 1000

func (v *Vault) rebuildStaleProjections[K, V any](
	ctx context.Context,
	collection Name,
	projections attachedProjections[K, V],
	storedRecordsIn func(*Txn) storedRecords,
) error {
	versions, err := v.projectionVersions()
	if err != nil {
		return err
	}
	stale, err := v.staleProjectionsAmong(ctx, versions, projections.buckets())
	if err != nil || len(stale) == 0 {
		return err
	}
	rebuildBegan := time.Now()
	reportRebuildBegan(ctx, collection, stale)
	if err := v.rebuildProjections(
		ctx,
		collection,
		projections,
		storedRecordsIn,
		stale,
	); err != nil {
		return err
	}
	if err := v.recordProjectionVersions(ctx, versions, stale); err != nil {
		return err
	}
	reportRebuilt(ctx, collection, stale, time.Since(rebuildBegan))

	return nil
}

func (v *Vault) staleProjectionsAmong(
	ctx context.Context,
	versions projectionVersions,
	buckets []projectionBucket,
) ([]projectionBucket, error) {
	var stale []projectionBucket
	if err := v.View(ctx, func(tx *Txn) error {
		var err error
		stale, err = versions.staleAmong(tx, buckets)

		return err
	}); err != nil {
		return nil, fmt.Errorf("read projection versions: %w", err)
	}

	return stale, nil
}

func reportRebuildBegan(ctx context.Context, collection Name, stale []projectionBucket) {
	slog.InfoContext(
		ctx,
		"vault projections rebuild began",
		slog.String("collection", string(collection)),
		slog.Any("projections", bucketNamesOf(stale)),
	)
}

func bucketNamesOf(buckets []projectionBucket) []Name {
	names := make([]Name, 0, len(buckets))
	for _, bucket := range buckets {
		names = append(names, bucket.bucketName())
	}

	return names
}

func (v *Vault) rebuildProjections[K, V any](
	ctx context.Context,
	collection Name,
	projections attachedProjections[K, V],
	storedRecordsIn func(*Txn) storedRecords,
	stale []projectionBucket,
) error {
	for _, bucket := range stale {
		if err := v.updateInBatches(ctx, bucket.clearFrom); err != nil {
			return fmt.Errorf("clear stale projection: %w", err)
		}
	}
	progress, err := v.rebuildProgressOf(ctx, collection, storedRecordsIn)
	if err != nil {
		return err
	}
	if err := v.updateInBatches(ctx, func(tx *Txn, firstKey []byte, limit int) ([]byte, error) {
		tx.RunAfterCommit(func() { progress.batchProjected(ctx) })

		return projections.projectStoredRecordsFrom(tx, storedRecordsIn(tx), firstKey, limit, stale)
	}); err != nil {
		return fmt.Errorf("rebuild stale projections of %s: %w", collection, err)
	}

	return nil
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
	versions projectionVersions,
	rebuilt []projectionBucket,
) error {
	if err := v.Update(ctx, func(tx *Txn) error {
		return versions.record(tx, rebuilt)
	}); err != nil {
		return fmt.Errorf("record projection versions: %w", err)
	}

	return nil
}

func reportRebuilt(
	ctx context.Context,
	collection Name,
	rebuilt []projectionBucket,
	took time.Duration,
) {
	slog.InfoContext(
		ctx,
		"vault projections rebuilt",
		slog.String("collection", string(collection)),
		slog.Any("projections", bucketNamesOf(rebuilt)),
		slog.Duration("took", took),
	)
}
