package nodeconfiguration

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvDistributionEnabled               = "YACY_DISTRIBUTION_ENABLED"
	EnvDistributionRedundancy            = "YACY_DISTRIBUTION_REDUNDANCY"
	EnvDistributionPartitionExponent     = "YACY_DISTRIBUTION_PARTITION_EXPONENT"
	EnvDistributionPostingsPerBatch      = "YACY_DISTRIBUTION_POSTINGS_PER_BATCH"
	EnvDistributionCycleInterval         = "YACY_DISTRIBUTION_CYCLE_INTERVAL"
	EnvDistributionDrainBudget           = "YACY_DISTRIBUTION_DRAIN_BUDGET"
	EnvDistributionLongestOfferInterval  = "YACY_DISTRIBUTION_LONGEST_OFFER_INTERVAL"
	EnvDistributionShortestOfferInterval = "YACY_DISTRIBUTION_SHORTEST_OFFER_INTERVAL"
	EnvDistributionRecipientCooldown     = "YACY_DISTRIBUTION_RECIPIENT_COOLDOWN"
	EnvDistributionMinReachablePeers     = "YACY_DISTRIBUTION_MIN_REACHABLE_PEERS"
	EnvDistributionURLMetadataBatchSize  = "YACY_DISTRIBUTION_URL_METADATA_BATCH_SIZE"

	DefaultDistributionEnabled               = false
	DefaultDistributionRedundancy            = 3
	DefaultDistributionPartitionExponent     = 4
	DefaultDistributionPostingsPerBatch      = 1000
	DefaultDistributionCycleInterval         = time.Minute
	DefaultDistributionDrainBudget           = time.Minute
	DefaultDistributionLongestOfferInterval  = 24 * time.Hour
	DefaultDistributionShortestOfferInterval = 5 * time.Minute
	DefaultDistributionRecipientCooldown     = 10 * time.Minute
	DefaultDistributionMinReachablePeers     = 32
	DefaultDistributionURLMetadataBatchSize  = 50
)

type DistributionConfig struct {
	Enabled              bool
	Redundancy           int
	PartitionExponent    uint
	PostingsPerBatch     int
	CycleInterval        time.Duration
	DrainBudget          time.Duration
	OfferInterval        OfferIntervalConfig
	RecipientCooldown    time.Duration
	MinReachablePeers    int
	URLMetadataBatchSize int
}

func loadDistributionConfig(getenv func(string) string) (DistributionConfig, error) {
	enabled, err := envconfig.Bool(getenv, EnvDistributionEnabled, DefaultDistributionEnabled)
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

	cycleInterval, err := envconfig.Duration(
		getenv, EnvDistributionCycleInterval, DefaultDistributionCycleInterval,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	drainBudget, err := envconfig.Duration(
		getenv, EnvDistributionDrainBudget, DefaultDistributionDrainBudget,
	)
	if err != nil {
		return DistributionConfig{}, err
	}

	offerInterval, err := loadOfferIntervalConfig(getenv)
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
		Redundancy:           redundancy,
		PartitionExponent:    uint(partitionExponent),
		PostingsPerBatch:     postingsPerBatch,
		CycleInterval:        cycleInterval,
		DrainBudget:          drainBudget,
		OfferInterval:        offerInterval,
		RecipientCooldown:    recipientCooldown,
		MinReachablePeers:    minReachablePeers,
		URLMetadataBatchSize: urlMetadataBatchSize,
	}, nil
}

type OfferIntervalConfig struct {
	Longest  time.Duration
	Shortest time.Duration
}

func loadOfferIntervalConfig(getenv func(string) string) (OfferIntervalConfig, error) {
	longest, err := envconfig.Duration(
		getenv, EnvDistributionLongestOfferInterval, DefaultDistributionLongestOfferInterval,
	)
	if err != nil {
		return OfferIntervalConfig{}, err
	}

	shortest, err := envconfig.Duration(
		getenv, EnvDistributionShortestOfferInterval, DefaultDistributionShortestOfferInterval,
	)
	if err != nil {
		return OfferIntervalConfig{}, err
	}

	return OfferIntervalConfig{Longest: longest, Shortest: shortest}, nil
}
