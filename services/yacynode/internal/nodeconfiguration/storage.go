package nodeconfiguration

import (
	"path/filepath"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvDataDir      = "YACY_DATA_DIR"
	EnvStorageQuota = "YACY_STORAGE_QUOTA"

	DefaultDataDir = "./data"
	DefaultQuota   = "1GB"

	StorageFileName = "yacy-rwipostings.db"
)

type StorageConfig struct {
	Path      string
	QuotaByte int64
}

func loadStorageConfig(getenv func(string) string) (StorageConfig, error) {
	quota, err := envconfig.ByteSize(getenv, EnvStorageQuota, DefaultQuota)
	if err != nil {
		return StorageConfig{}, err
	}

	dataDir := envconfig.String(getenv, EnvDataDir, DefaultDataDir)

	return StorageConfig{
		Path:      filepath.Join(dataDir, StorageFileName),
		QuotaByte: quota,
	}, nil
}
