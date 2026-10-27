package yacypeerassembly

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	EnvDataDirectory          = "YACYPEER_DATA_DIR"
	EnvNetworkName            = "YACYPEER_NETWORK_NAME"
	EnvSeedlistURLs           = "YACYPEER_SEEDLIST_URLS"
	EnvOpsAddr                = "YACYPEER_OPS_ADDR"
	EnvPeerProxyURL           = "YACYPEER_PEER_PROXY_URL"
	EnvPageFetchProxyURL      = "YACYPEER_PAGE_FETCH_PROXY_URL"
	EnvMetricHistoryRetention = "YACYPEER_METRIC_HISTORY_RETENTION"

	DefaultDataDirectory          = "data"
	DefaultOpsAddr                = loopback.Host + ":9090"
	DefaultMetricHistoryRetention = 7 * 24 * time.Hour
	DefaultSeedlistURLs           = "http://sixcooler.de/yacy/seed.txt," +
		"https://sonst.mifritscher.de/yacy/seed.txt," +
		"http://5.45.105.16/yacyseed," +
		"http://yacy.v16.de/seed/seed.txt," +
		"https://frank-siebert.de/seed.txt," +
		"http://seedlist.wertewesten.net/seed.txt"
)

var proxyEnvironment = []string{
	"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy", "NO_PROXY", "no_proxy",
}

var pageFetchProxySchemes = []string{"http", "https", "socks5", "socks5h"}

var (
	errProxyEnvironment = errors.New(
		"must not be set; name the proxy in " + EnvPeerProxyURL + " or " + EnvPageFetchProxyURL,
	)
	errPageFetchProxy = errors.New(
		"must be an absolute URL with scheme " + strings.Join(pageFetchProxySchemes, ", "),
	)
)

type PeerConfig struct {
	PeerProxyURL           *url.URL
	PageFetchProxyURL      *url.URL
	DataDirectory          string
	NetworkName            string
	SeedlistURLs           string
	OpsAddr                string
	MetricHistoryRetention time.Duration
}

func LoadPeerConfig(getenv func(string) string) (PeerConfig, error) {
	if err := requireNoProxyEnvironment(getenv); err != nil {
		return PeerConfig{}, err
	}
	networkName := envconfig.String(getenv, EnvNetworkName, yacyproto.DefaultNetwork)
	seedlistURLs, err := seedlistURLsFor(getenv, networkName)
	if err != nil {
		return PeerConfig{}, err
	}
	peerProxyURL, err := peerProxyURLFrom(getenv)
	if err != nil {
		return PeerConfig{}, err
	}
	pageFetchProxyURL, err := pageFetchProxyURLFrom(getenv)
	if err != nil {
		return PeerConfig{}, err
	}
	metricHistoryRetention, err := envconfig.Duration(
		getenv,
		EnvMetricHistoryRetention,
		DefaultMetricHistoryRetention,
	)
	if err != nil {
		return PeerConfig{}, err
	}
	return PeerConfig{
		PeerProxyURL:           peerProxyURL,
		PageFetchProxyURL:      pageFetchProxyURL,
		DataDirectory:          envconfig.String(getenv, EnvDataDirectory, DefaultDataDirectory),
		NetworkName:            networkName,
		SeedlistURLs:           seedlistURLs,
		OpsAddr:                envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		MetricHistoryRetention: metricHistoryRetention,
	}, nil
}

func requireNoProxyEnvironment(getenv func(string) string) error {
	var failures []error
	for _, key := range proxyEnvironment {
		if getenv(key) != "" {
			failures = append(failures, fmt.Errorf("%s: %w", key, errProxyEnvironment))
		}
	}
	return errors.Join(failures...)
}

func seedlistURLsFor(getenv func(string) string, networkName string) (string, error) {
	if networkName == yacyproto.DefaultNetwork {
		return envconfig.String(getenv, EnvSeedlistURLs, DefaultSeedlistURLs), nil
	}
	return envconfig.Required(getenv, EnvSeedlistURLs)
}

func peerProxyURLFrom(getenv func(string) string) (*url.URL, error) {
	if strings.TrimSpace(getenv(EnvPeerProxyURL)) == "" {
		return nil, nil
	}
	return envconfig.RequiredHTTPURL(getenv, EnvPeerProxyURL)
}

func pageFetchProxyURLFrom(getenv func(string) string) (*url.URL, error) {
	rawURL := strings.TrimSpace(getenv(EnvPageFetchProxyURL))
	if rawURL == "" {
		return nil, nil
	}
	proxyURL, err := url.Parse(rawURL)
	if err != nil || proxyURL.Host == "" ||
		!slices.Contains(pageFetchProxySchemes, proxyURL.Scheme) {
		return nil, fmt.Errorf("%s: %w", EnvPageFetchProxyURL, errPageFetchProxy)
	}
	return proxyURL, nil
}

func (cfg PeerConfig) pageFetchProxyURLOr(egressProxyURL *url.URL) *url.URL {
	if cfg.PageFetchProxyURL != nil {
		return cfg.PageFetchProxyURL
	}
	return egressProxyURL
}

func (cfg PeerConfig) namesAPeerProxy() bool {
	return cfg.PeerProxyURL != nil
}

func (cfg PeerConfig) directoryOf(serviceName string) string {
	return filepath.Join(cfg.DataDirectory, serviceName)
}
