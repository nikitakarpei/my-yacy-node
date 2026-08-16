package nodeconfiguration

import "github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"

const (
	EnvEscrowPostingCapacity = "YACY_ESCROW_POSTING_CAPACITY"

	DefaultEscrowPostingCapacity = 8192
)

type EscrowConfig struct {
	PostingCapacity int
}

func loadEscrowConfig(getenv func(string) string) (EscrowConfig, error) {
	postingCapacity, err := envconfig.PositiveInt(
		getenv,
		EnvEscrowPostingCapacity,
		DefaultEscrowPostingCapacity,
	)
	if err != nil {
		return EscrowConfig{}, err
	}

	return EscrowConfig{PostingCapacity: postingCapacity}, nil
}
