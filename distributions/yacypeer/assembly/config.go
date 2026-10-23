package yacypeerassembly

import (
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	EnvDataDirectory = "YACYPEER_DATA_DIR"
	EnvNetworkName   = "YACYPEER_NETWORK_NAME"
	EnvSeedlistURLs  = "YACYPEER_SEEDLIST_URLS"
	EnvOpsAddr       = "YACYPEER_OPS_ADDR"

	DefaultDataDirectory = "data"
	DefaultOpsAddr       = ":9090"
)

type PeerConfig struct {
	DataDirectory string
	NetworkName   string
	SeedlistURLs  string
	OpsAddr       string
}

func LoadPeerConfig(getenv func(string) string) (PeerConfig, error) {
	seedlistURLs, err := envconfig.Required(getenv, EnvSeedlistURLs)
	if err != nil {
		return PeerConfig{}, err
	}
	return PeerConfig{
		DataDirectory: envconfig.String(getenv, EnvDataDirectory, DefaultDataDirectory),
		NetworkName:   envconfig.String(getenv, EnvNetworkName, yacyproto.DefaultNetwork),
		SeedlistURLs:  seedlistURLs,
		OpsAddr:       envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}
