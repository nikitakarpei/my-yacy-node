package vault

import (
	"context"
	"fmt"
	"log/slog"
)

const batchesPerProgressReport = 100

type rebuildProgress struct {
	collection       Name
	recordAmount     int
	projectedBatches int
}

func (v *Vault) rebuildProgressOf(
	ctx context.Context,
	collection Name,
	storedRecordsIn func(*Txn) storedRecords,
) (*rebuildProgress, error) {
	progress := &rebuildProgress{collection: collection}
	if err := v.view(ctx, func(tx *Txn) error {
		recordAmount, err := storedRecordsIn(tx).Len()
		progress.recordAmount = recordAmount

		return err
	}); err != nil {
		return nil, fmt.Errorf("count records to project: %w", err)
	}

	return progress, nil
}

func (p *rebuildProgress) batchProjected(ctx context.Context) {
	p.projectedBatches++
	if p.projectedBatches%batchesPerProgressReport != 0 {
		return
	}
	slog.InfoContext(
		ctx,
		"vault projections rebuild progressed",
		slog.String("collection", string(p.collection)),
		slog.Int("projectedRecords", p.projectedBatches*rebuildBatchRecords),
		slog.Int("records", p.recordAmount),
	)
}
