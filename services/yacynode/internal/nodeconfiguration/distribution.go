package nodeconfiguration

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvDistributionEnabled                     = "YACY_DISTRIBUTION_ENABLED"
	EnvDistributionHandoffEnabled              = "YACY_DISTRIBUTION_HANDOFF_ENABLED"
	EnvDistributionRedundancy                  = "YACY_DISTRIBUTION_REDUNDANCY"
	EnvDistributionPartitionExponent           = "YACY_DISTRIBUTION_PARTITION_EXPONENT"
	EnvDistributionPostingsPerBatch            = "YACY_DISTRIBUTION_POSTINGS_PER_BATCH"
	EnvDistributionWorkers                     = "YACY_DISTRIBUTION_WORKERS"
	EnvDistributionRescanDelay                 = "YACY_DISTRIBUTION_RESCAN_DELAY"
	EnvDistributionRefreshInterval             = "YACY_DISTRIBUTION_REFRESH_INTERVAL"
	EnvDistributionRecipientCooldown           = "YACY_DISTRIBUTION_RECIPIENT_COOLDOWN"
	EnvDistributionFailedOffersBeforeExclusion = "YACY_DISTRIBUTION_FAILED_OFFERS_BEFORE_EXCLUSION"
	EnvDistributionPeerExclusionDuration       = "YACY_DISTRIBUTION_PEER_EXCLUSION_DURATION"
	EnvDistributionMinReachablePeers           = "YACY_DISTRIBUTION_MIN_REACHABLE_PEERS"
	EnvDistributionURLMetadataBatchSize        = "YACY_DISTRIBUTION_URL_METADATA_BATCH_SIZE"

	DefaultDistributionEnabled                     = false
	DefaultDistributionHandoffEnabled              = true
	DefaultDistributionRedundancy                  = 3
	DefaultDistributionPartitionExponent           = 4
	DefaultDistributionPostingsPerBatch            = 1000
	DefaultDistributionWorkers                     = 2
	DefaultDistributionRescanDelay                 = 10 * time.Second
	DefaultDistributionRefreshInterval             = 24 * time.Hour
	DefaultDistributionRecipientCooldown           = 10 * time.Minute
	DefaultDistributionFailedOffersBeforeExclusion = 3
	DefaultDistributionPeerExclusionDuration       = time.Hour
	DefaultDistributionMinReachablePeers           = 32
	DefaultDistributionURLMetadataBatchSize        = 50
)

type DistributionConfig struct {
	Enabled                     bool
	HandoffEnabled              bool
	Redundancy                  int
	PartitionExponent           uint
	PostingsPerBatch            int
	AmountOfWorkers             int
	RescanDelay                 time.Duration
	RefreshInterval             time.Duration
	RecipientCooldown           time.Duration
	FailedOffersBeforeExclusion int
	PeerExclusionDuration       time.Duration
	MinReachablePeers           int
	URLMetadataBatchSize        int
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

	amountOfWorkers, err := envconfig.PositiveInt(
		getenv, EnvDistributionWorkers, DefaultDistributionWorkers,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	rescanDelay, err := envconfig.Duration(
		getenv, EnvDistributionRescanDelay, DefaultDistributionRescanDelay,
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

	failedOffersBeforeExclusion, err := envconfig.PositiveInt(
		getenv,
		EnvDistributionFailedOffersBeforeExclusion,
		DefaultDistributionFailedOffersBeforeExclusion,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	peerExclusionDuration, err := envconfig.Duration(
		getenv,
		EnvDistributionPeerExclusionDuration,
		DefaultDistributionPeerExclusionDuration,
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
		Enabled:                     enabled,
		HandoffEnabled:              handoffEnabled,
		Redundancy:                  redundancy,
		PartitionExponent:           uint(partitionExponent),
		PostingsPerBatch:            postingsPerBatch,
		AmountOfWorkers:             amountOfWorkers,
		RescanDelay:                 rescanDelay,
		RefreshInterval:             refreshInterval,
		RecipientCooldown:           recipientCooldown,
		FailedOffersBeforeExclusion: failedOffersBeforeExclusion,
		PeerExclusionDuration:       peerExclusionDuration,
		MinReachablePeers:           minReachablePeers,
		URLMetadataBatchSize:        urlMetadataBatchSize,
	}, nil
}
