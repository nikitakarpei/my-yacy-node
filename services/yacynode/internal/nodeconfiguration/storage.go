package nodeconfiguration

import (
	"path/filepath"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvDataDir               = "YACY_DATA_DIR"
	EnvStorageQuota          = "YACY_STORAGE_QUOTA"
	EnvBBoltWriteBatchWrites = "YACY_BBOLT_WRITE_BATCH_WRITES"
	EnvBBoltWriteBatchDelay  = "YACY_BBOLT_WRITE_BATCH_DELAY"

	DefaultDataDir                      = "./data"
	DefaultQuota                        = "1GB"
	DefaultBBoltWriteBatchMaximumWrites = 1000
	DefaultBBoltWriteBatchMaximumDelay  = 10 * time.Millisecond

	StorageFileName = "node.db"
)

type StorageConfig struct {
	Path                         string
	QuotaByte                    int64
	BBoltWriteBatchMaximumWrites int
	BBoltWriteBatchMaximumDelay  time.Duration
}

func loadStorageConfig(getenv func(string) string) (StorageConfig, error) {
	quota, err := envconfig.ByteSize(getenv, EnvStorageQuota, DefaultQuota)
	if err != nil {
		return StorageConfig{}, err
	}

	writeBatchWrites, err := envconfig.PositiveInt(
		getenv,
		EnvBBoltWriteBatchWrites,
		DefaultBBoltWriteBatchMaximumWrites,
	)
	if err != nil {
		return StorageConfig{}, err
	}

	writeBatchDelay, err := envconfig.NonNegativeDuration(
		getenv,
		EnvBBoltWriteBatchDelay,
		DefaultBBoltWriteBatchMaximumDelay,
	)
	if err != nil {
		return StorageConfig{}, err
	}

	dataDir := envconfig.String(getenv, EnvDataDir, DefaultDataDir)

	return StorageConfig{
		Path:                         filepath.Join(dataDir, StorageFileName),
		QuotaByte:                    quota,
		BBoltWriteBatchMaximumWrites: writeBatchWrites,
		BBoltWriteBatchMaximumDelay:  writeBatchDelay,
	}, nil
}
