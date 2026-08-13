package bolt

import (
	"context"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

func (e *engine) QuotaBytes() int64 {
	return e.quotaBytes
}

func (e *engine) UsedBytes(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("context: %w", err)
	}

	var used int64
	if err := e.db.View(func(tx *bolt.Tx) error {
		stats := e.db.Stats()
		pageSize := int64(e.db.Info().PageSize)
		free := int64(stats.FreePageN+stats.PendingPageN) * pageSize
		used = tx.Size() - free

		return nil
	}); err != nil {
		return 0, fmt.Errorf("read storage stats: %w", err)
	}
	if used < 0 {
		used = 0
	}

	return used, nil
}
