package yacypeerassembly

import (
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	EnvDataDirectory  = "YACYPEER_DATA_DIR"
	EnvNetworkName    = "YACYPEER_NETWORK_NAME"
	EnvSeedlistURLs   = "YACYPEER_SEEDLIST_URLS"
	EnvOpsAddr        = "YACYPEER_OPS_ADDR"
	EnvEgressProxyURL = "YACYPEER_EGRESS_PROXY_URL"

	DefaultDataDirectory = "data"
	DefaultOpsAddr       = loopback.Host + ":9090"
)

type PeerConfig struct {
	EgressProxyURL *url.URL
	DataDirectory  string
	NetworkName    string
	SeedlistURLs   string
	OpsAddr        string
}

func LoadPeerConfig(getenv func(string) string) (PeerConfig, error) {
	seedlistURLs, err := envconfig.Required(getenv, EnvSeedlistURLs)
	if err != nil {
		return PeerConfig{}, err
	}
	egressProxyURL, err := egressProxyURLFrom(getenv)
	if err != nil {
		return PeerConfig{}, err
	}
	return PeerConfig{
		EgressProxyURL: egressProxyURL,
		DataDirectory:  envconfig.String(getenv, EnvDataDirectory, DefaultDataDirectory),
		NetworkName:    envconfig.String(getenv, EnvNetworkName, yacyproto.DefaultNetwork),
		SeedlistURLs:   seedlistURLs,
		OpsAddr:        envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}

func egressProxyURLFrom(getenv func(string) string) (*url.URL, error) {
	if strings.TrimSpace(getenv(EnvEgressProxyURL)) == "" {
		return nil, nil
	}
	return envconfig.RequiredHTTPURL(getenv, EnvEgressProxyURL)
}

func (cfg PeerConfig) namesAnEgressProxy() bool {
	return cfg.EgressProxyURL != nil
}
