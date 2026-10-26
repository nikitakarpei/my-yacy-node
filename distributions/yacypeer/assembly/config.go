package yacypeerassembly

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilnode"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	EnvDataDirectory     = "YACYPEER_DATA_DIR"
	EnvNetworkName       = "YACYPEER_NETWORK_NAME"
	EnvSeedlistURLs      = "YACYPEER_SEEDLIST_URLS"
	EnvOpsAddr           = "YACYPEER_OPS_ADDR"
	EnvEgressProxyURL    = "YACYPEER_EGRESS_PROXY_URL"
	EnvYggdrasilPeers    = "YACYPEER_YGGDRASIL_PEERS"
	EnvYggdrasilListen   = "YACYPEER_YGGDRASIL_LISTEN"
	EnvYggdrasilPeerList = "YACYPEER_YGGDRASIL_PEER_LIST"

	DefaultDataDirectory = "data"
	DefaultOpsAddr       = loopback.Host + ":9090"
	DefaultSeedlistURLs  = "http://sixcooler.de/yacy/seed.txt," +
		"https://sonst.mifritscher.de/yacy/seed.txt," +
		"http://5.45.105.16/yacyseed," +
		"http://yacy.v16.de/seed/seed.txt," +
		"https://frank-siebert.de/seed.txt," +
		"http://seedlist.wertewesten.net/seed.txt"
)

var errNoScheme = errors.New("must include a scheme")

type PeerConfig struct {
	EgressProxyURL    *url.URL
	YggdrasilPeering  yggdrasilnode.Peering
	YggdrasilPeerList *url.URL
	DataDirectory     string
	NetworkName       string
	SeedlistURLs      string
	OpsAddr           string
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
	yggdrasilPeering, err := yggdrasilPeeringFrom(getenv)
	if err != nil {
		return PeerConfig{}, err
	}
	yggdrasilPeerList, err := yggdrasilPeerListFrom(getenv)
	if err != nil {
		return PeerConfig{}, err
	}
	cfg := PeerConfig{
		EgressProxyURL:    egressProxyURL,
		YggdrasilPeering:  yggdrasilPeering,
		YggdrasilPeerList: yggdrasilPeerList,
		DataDirectory:     envconfig.String(getenv, EnvDataDirectory, DefaultDataDirectory),
		NetworkName:       networkName,
		SeedlistURLs:      seedlistURLs,
		OpsAddr:           envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}
	if cfg.joinsYggdrasil() && cfg.namesAnEgressProxy() {
		return PeerConfig{}, fmt.Errorf(
			"%s: must not be set when %s, %s or %s is set",
			EnvEgressProxyURL,
			EnvYggdrasilPeers,
			EnvYggdrasilListen,
			EnvYggdrasilPeerList,
		)
	}
	return cfg, nil
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

func yggdrasilPeeringFrom(getenv func(string) string) (yggdrasilnode.Peering, error) {
	peerURLs, err := urlsFrom(getenv, EnvYggdrasilPeers)
	if err != nil {
		return yggdrasilnode.Peering{}, err
	}
	listenURLs, err := urlsFrom(getenv, EnvYggdrasilListen)
	if err != nil {
		return yggdrasilnode.Peering{}, err
	}
	return yggdrasilnode.Peering{PeerURLs: peerURLs, ListenURLs: listenURLs}, nil
}

func yggdrasilPeerListFrom(getenv func(string) string) (*url.URL, error) {
	if strings.TrimSpace(getenv(EnvYggdrasilPeerList)) == "" {
		return nil, nil
	}
	return envconfig.RequiredHTTPURL(getenv, EnvYggdrasilPeerList)
}

func urlsFrom(getenv func(string) string, key string) ([]*url.URL, error) {
	var urls []*url.URL
	for _, value := range envconfig.List(getenv, key) {
		parsed, err := url.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		if parsed.Scheme == "" {
			return nil, fmt.Errorf("%s: %q %w", key, value, errNoScheme)
		}
		urls = append(urls, parsed)
	}
	return urls, nil
}

func (cfg PeerConfig) joinsYggdrasil() bool {
	return len(cfg.YggdrasilPeering.PeerURLs)+len(cfg.YggdrasilPeering.ListenURLs) > 0 ||
		cfg.choosesYggdrasilPeers()
}

func (cfg PeerConfig) choosesYggdrasilPeers() bool {
	return cfg.YggdrasilPeerList != nil
}
