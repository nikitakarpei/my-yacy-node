// Package eviction frees storage when the vault nears its quota. It owns no
// buckets: it reads usage from the storage kernel, names the stalest URLs, and
// purges their metadata within one capacity-exempt transaction, so every
// collection that follows url metadata clears atomically without sharing a
// schema.
package eviction

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmetastaleness"
)

type Config struct {
	TargetFraction  float64
	URLsPerBatch    int
	BatchesPerSweep int
}

type Result struct {
	URLsDeleted int
}

type Sweeper interface {
	Sweep(ctx context.Context) (Result, error)
}

func NewSweeper(
	vault *vault.Vault,
	urls urlmeta.URLEvictor,
	stale urlmetastaleness.StaleURLSource,
	cfg Config,
) Sweeper {
	return quotaSweeper{
		vault:           vault,
		urls:            urls,
		stale:           stale,
		target:          cfg.TargetFraction,
		urlsPerBatch:    cfg.URLsPerBatch,
		batchesPerSweep: cfg.BatchesPerSweep,
	}
}
