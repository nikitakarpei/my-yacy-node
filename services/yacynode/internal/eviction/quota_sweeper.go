package eviction

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmetastaleness"
)

const boundReachedMessage = "storage eviction stopped at its sweep bound"

type quotaSweeper struct {
	vault           *vault.Vault
	urls            urlmeta.URLEvictor
	stale           urlmetastaleness.StaleURLSource
	target          float64
	urlsPerBatch    int
	batchesPerSweep int
}

func (s quotaSweeper) Sweep(ctx context.Context) (Result, error) {
	quota := s.vault.QuotaBytes()
	if quota <= 0 || s.urlsPerBatch <= 0 || s.batchesPerSweep <= 0 {
		return Result{}, nil
	}
	highWater := int64(float64(quota) * s.target)

	var total Result
	for range s.batchesPerSweep {
		used, err := s.vault.UsedBytes(ctx)
		if err != nil {
			return total, fmt.Errorf("measure usage: %w", err)
		}
		if used < highWater {
			return total, nil
		}

		batch, err := s.purgeStalest(ctx)
		if err != nil {
			return total, err
		}
		total.URLsDeleted += batch.URLsDeleted
		if batch.URLsDeleted == 0 {
			return total, nil
		}
	}

	slog.WarnContext(ctx, boundReachedMessage, slog.Int("urls", total.URLsDeleted))

	return total, nil
}

func (s quotaSweeper) purgeStalest(ctx context.Context) (Result, error) {
	var result Result
	err := s.vault.Update(ctx, func(tx *vault.Txn) error {
		stalest, err := s.stale.StalestURLs(tx, s.urlsPerBatch)
		if err != nil {
			return fmt.Errorf("select stale urls: %w", err)
		}

		purged, err := s.urls.Purge(ctx, tx, stalest)
		if err != nil {
			return fmt.Errorf("purge urls: %w", err)
		}
		result = Result{URLsDeleted: purged.URLsDeleted}

		return nil
	})
	if err != nil {
		return Result{}, fmt.Errorf("purge batch: %w", err)
	}

	return result, nil
}
