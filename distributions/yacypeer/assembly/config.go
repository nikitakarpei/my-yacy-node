package yacypeerassembly

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	yacynodeassembly "github.com/nikitakarpei/yacy-rwi-node/yacynode/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
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
	DefaultSeedlistURLs  = "http://sixcooler.de/yacy/seed.txt," +
		"https://sonst.mifritscher.de/yacy/seed.txt," +
		"http://5.45.105.16/yacyseed," +
		"http://yacy.v16.de/seed/seed.txt," +
		"https://frank-siebert.de/seed.txt," +
		"http://seedlist.wertewesten.net/seed.txt"
)

type PeerConfig struct {
	EgressProxyURL *url.URL
	DataDirectory  string
	NetworkName    string
	SeedlistURLs   string
	OpsAddr        string
}

func LoadPeerConfig(getenv func(string) string) (PeerConfig, error) {
	networkName := envconfig.String(getenv, EnvNetworkName, yacyproto.DefaultNetwork)
	seedlistURLs, err := seedlistURLsFor(getenv, networkName)
	if err != nil {
		return PeerConfig{}, err
	}
	egressProxyURL, err := egressProxyURLFrom(getenv)
	if err != nil {
		return PeerConfig{}, err
	}
	if err := requireAdvertiseHostBehind(egressProxyURL, getenv); err != nil {
		return PeerConfig{}, err
	}
	return PeerConfig{
		EgressProxyURL: egressProxyURL,
		DataDirectory:  envconfig.String(getenv, EnvDataDirectory, DefaultDataDirectory),
		NetworkName:    networkName,
		SeedlistURLs:   seedlistURLs,
		OpsAddr:        envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}

func seedlistURLsFor(getenv func(string) string, networkName string) (string, error) {
	if networkName == yacyproto.DefaultNetwork {
		return envconfig.String(getenv, EnvSeedlistURLs, DefaultSeedlistURLs), nil
	}
	return envconfig.Required(getenv, EnvSeedlistURLs)
}

func egressProxyURLFrom(getenv func(string) string) (*url.URL, error) {
	if strings.TrimSpace(getenv(EnvEgressProxyURL)) == "" {
		return nil, nil
	}
	return envconfig.RequiredHTTPURL(getenv, EnvEgressProxyURL)
}

func requireAdvertiseHostBehind(egressProxyURL *url.URL, getenv func(string) string) error {
	if egressProxyURL == nil {
		return nil
	}
	nodeEnvironment, err := serviceenvironment.For(yacynodeService, nil, nil, nil, getenv)
	if err != nil {
		return err
	}
	if strings.TrimSpace(nodeEnvironment.Lookup(yacynodeassembly.EnvAdvertiseHost)) == "" {
		return fmt.Errorf(
			"%s: must be set when %s is set",
			yacynodeassembly.EnvAdvertiseHost,
			EnvEgressProxyURL,
		)
	}
	return nil
}

func (cfg PeerConfig) namesAnEgressProxy() bool {
	return cfg.EgressProxyURL != nil
}
