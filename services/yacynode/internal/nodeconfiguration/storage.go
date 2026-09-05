package nodeconfiguration

import (
	"path/filepath"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvDataDir                     = "YACY_DATA_DIR"
	EnvStorageQuota                = "YACY_STORAGE_QUOTA"
	EnvPebbleBlockCache            = "YACY_PEBBLE_BLOCK_CACHE"
	EnvPebbleMemtableSize          = "YACY_PEBBLE_MEMTABLE_SIZE"
	EnvPebbleCompactionConcurrency = "YACY_PEBBLE_COMPACTION_CONCURRENCY"
	EnvPebbleOpenFileLimit         = "YACY_PEBBLE_OPEN_FILE_LIMIT"

	DefaultDataDir                     = "./data"
	DefaultQuota                       = "1GB"
	DefaultPebbleBlockCache            = "64MB"
	DefaultPebbleMemtableSize          = "8MB"
	DefaultPebbleCompactionConcurrency = 1
	DefaultPebbleOpenFileLimit         = 1000

	StorageDirectoryName = "node"
)

type StorageConfig struct {
	Path                  string
	QuotaByte             int64
	BlockCacheByte        int64
	MemtableByte          int64
	CompactionConcurrency int
	OpenFileLimit         int
}

func loadStorageConfig(getenv func(string) string) (StorageConfig, error) {
	quota, err := envconfig.ByteSize(getenv, EnvStorageQuota, DefaultQuota)
	if err != nil {
		return StorageConfig{}, err
	}

	blockCache, err := envconfig.ByteSize(getenv, EnvPebbleBlockCache, DefaultPebbleBlockCache)
	if err != nil {
		return StorageConfig{}, err
	}

	memtable, err := envconfig.ByteSize(getenv, EnvPebbleMemtableSize, DefaultPebbleMemtableSize)
	if err != nil {
		return StorageConfig{}, err
	}

	compactionConcurrency, err := envconfig.PositiveInt(
		getenv,
		EnvPebbleCompactionConcurrency,
		DefaultPebbleCompactionConcurrency,
	)
	if err != nil {
		return StorageConfig{}, err
	}

	openFileLimit, err := envconfig.PositiveInt(
		getenv,
		EnvPebbleOpenFileLimit,
		DefaultPebbleOpenFileLimit,
	)
	if err != nil {
		return StorageConfig{}, err
	}

	dataDir := envconfig.String(getenv, EnvDataDir, DefaultDataDir)

	return StorageConfig{
		Path:                  filepath.Join(dataDir, StorageDirectoryName),
		QuotaByte:             quota,
		BlockCacheByte:        blockCache,
		MemtableByte:          memtable,
		CompactionConcurrency: compactionConcurrency,
		OpenFileLimit:         openFileLimit,
	}, nil
}
