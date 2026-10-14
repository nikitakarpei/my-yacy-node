package nodeconfiguration

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvDistributionEnabled              = "YACY_DISTRIBUTION_ENABLED"
	EnvDistributionHandoffEnabled       = "YACY_DISTRIBUTION_HANDOFF_ENABLED"
	EnvDistributionRedundancy           = "YACY_DISTRIBUTION_REDUNDANCY"
	EnvDistributionPartitionExponent    = "YACY_DISTRIBUTION_PARTITION_EXPONENT"
	EnvDistributionPostingsPerBatch     = "YACY_DISTRIBUTION_POSTINGS_PER_BATCH"
	EnvDistributionBatchConcurrency     = "YACY_DISTRIBUTION_BATCH_CONCURRENCY"
	EnvDistributionSectorRescanDelay    = "YACY_DISTRIBUTION_SECTOR_RESCAN_DELAY"
	EnvDistributionRefreshInterval      = "YACY_DISTRIBUTION_REFRESH_INTERVAL"
	EnvDistributionRecipientCooldown    = "YACY_DISTRIBUTION_RECIPIENT_COOLDOWN"
	EnvDistributionMinReachablePeers    = "YACY_DISTRIBUTION_MIN_REACHABLE_PEERS"
	EnvDistributionURLMetadataBatchSize = "YACY_DISTRIBUTION_URL_METADATA_BATCH_SIZE"

	DefaultDistributionEnabled              = false
	DefaultDistributionHandoffEnabled       = true
	DefaultDistributionRedundancy           = 3
	DefaultDistributionPartitionExponent    = 4
	DefaultDistributionPostingsPerBatch     = 1000
	DefaultDistributionBatchConcurrency     = 2
	DefaultDistributionSectorRescanDelay    = 10 * time.Second
	DefaultDistributionRefreshInterval      = 24 * time.Hour
	DefaultDistributionRecipientCooldown    = 10 * time.Minute
	DefaultDistributionMinReachablePeers    = 32
	DefaultDistributionURLMetadataBatchSize = 50
)

type DistributionConfig struct {
	Enabled              bool
	HandoffEnabled       bool
	Redundancy           int
	PartitionExponent    uint
	PostingsPerBatch     int
	BatchConcurrency     int
	SectorRescanDelay    time.Duration
	RefreshInterval      time.Duration
	RecipientCooldown    time.Duration
	MinReachablePeers    int
	URLMetadataBatchSize int
}

//nolint:funlen // one flat read per distribution setting
func loadDistributionConfig(getenv func(string) string) (DistributionConfig, error) {
	enabled, err := envconfig.Bool(getenv, EnvDistributionEnabled, DefaultDistributionEnabled)
	if err != nil {
		return DistributionConfig{}, err
	}

	handoffEnabled, err := envconfig.Bool(
		getenv, EnvDistributionHandoffEnabled, DefaultDistributionHandoffEnabled,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	redundancy, err := envconfig.PositiveInt(
		getenv, EnvDistributionRedundancy, DefaultDistributionRedundancy,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	partitionExponent, err := envconfig.PositiveInt(
		getenv, EnvDistributionPartitionExponent, DefaultDistributionPartitionExponent,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	postingsPerBatch, err := envconfig.PositiveInt(
		getenv, EnvDistributionPostingsPerBatch, DefaultDistributionPostingsPerBatch,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	batchConcurrency, err := envconfig.PositiveInt(
		getenv, EnvDistributionBatchConcurrency, DefaultDistributionBatchConcurrency,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	sectorRescanDelay, err := envconfig.Duration(
		getenv, EnvDistributionSectorRescanDelay, DefaultDistributionSectorRescanDelay,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	refreshInterval, err := envconfig.Duration(
		getenv, EnvDistributionRefreshInterval, DefaultDistributionRefreshInterval,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	recipientCooldown, err := envconfig.Duration(
		getenv, EnvDistributionRecipientCooldown, DefaultDistributionRecipientCooldown,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	minReachablePeers, err := envconfig.PositiveInt(
		getenv, EnvDistributionMinReachablePeers, DefaultDistributionMinReachablePeers,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	urlMetadataBatchSize, err := envconfig.PositiveInt(
		getenv, EnvDistributionURLMetadataBatchSize, DefaultDistributionURLMetadataBatchSize,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	return DistributionConfig{
		Enabled:              enabled,
		HandoffEnabled:       handoffEnabled,
		Redundancy:           redundancy,
		PartitionExponent:    uint(partitionExponent),
		PostingsPerBatch:     postingsPerBatch,
		BatchConcurrency:     batchConcurrency,
		SectorRescanDelay:    sectorRescanDelay,
		RefreshInterval:      refreshInterval,
		RecipientCooldown:    recipientCooldown,
		MinReachablePeers:    minReachablePeers,
		URLMetadataBatchSize: urlMetadataBatchSize,
	}, nil
}
