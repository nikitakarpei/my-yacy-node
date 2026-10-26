package yacypeerassembly

import (
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	EnvDataDirectory    = "YACYPEER_DATA_DIR"
	EnvNetworkName      = "YACYPEER_NETWORK_NAME"
	EnvSeedlistURLs     = "YACYPEER_SEEDLIST_URLS"
	EnvOpsAddr          = "YACYPEER_OPS_ADDR"
	EnvEgressProxyURL   = "YACYPEER_EGRESS_PROXY_URL"
	EnvMetricsRetention = "YACYPEER_METRICS_RETENTION"

	DefaultDataDirectory    = "data"
	DefaultOpsAddr          = loopback.Host + ":9090"
	DefaultMetricsRetention = 7 * 24 * time.Hour
	DefaultSeedlistURLs     = "http://sixcooler.de/yacy/seed.txt," +
		"https://sonst.mifritscher.de/yacy/seed.txt," +
		"http://5.45.105.16/yacyseed," +
		"http://yacy.v16.de/seed/seed.txt," +
		"https://frank-siebert.de/seed.txt," +
		"http://seedlist.wertewesten.net/seed.txt"
)

type PeerConfig struct {
	EgressProxyURL   *url.URL
	DataDirectory    string
	NetworkName      string
	SeedlistURLs     string
	OpsAddr          string
	MetricsRetention time.Duration
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
	metricsRetention, err := envconfig.Duration(
		getenv,
		EnvMetricsRetention,
		DefaultMetricsRetention,
	)
	if err != nil {
		return PeerConfig{}, err
	}
	return PeerConfig{
		EgressProxyURL:   egressProxyURL,
		DataDirectory:    envconfig.String(getenv, EnvDataDirectory, DefaultDataDirectory),
		NetworkName:      networkName,
		SeedlistURLs:     seedlistURLs,
		OpsAddr:          envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		MetricsRetention: metricsRetention,
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

func (cfg PeerConfig) namesAnEgressProxy() bool {
	return cfg.EgressProxyURL != nil
}
