package yacypeerassembly_test

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	yacypeerassembly "github.com/nikitakarpei/yacy-rwi-node/yacypeer/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilnode"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadPeerConfigRequiresTheSeedlistsOfAnotherNetwork(t *testing.T) {
	_, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvNetworkName: "realm",
	}))
	if err == nil {
		t.Fatal("config loaded without the seedlists of another network, want an error")
	}
}

func TestLoadPeerConfigFallsBackToTheDefaultOfEverySetting(t *testing.T) {
	cfg, err := yacypeerassembly.LoadPeerConfig(envFrom(nil))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	want := yacypeerassembly.PeerConfig{
		DataDirectory:          yacypeerassembly.DefaultDataDirectory,
		NetworkName:            yacyproto.DefaultNetwork,
		SeedlistURLs:           yacypeerassembly.DefaultSeedlistURLs,
		OpsAddr:                yacypeerassembly.DefaultOpsAddr,
		MetricHistoryRetention: yacypeerassembly.DefaultMetricHistoryRetention,
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("config = %+v, want %+v", cfg, want)
	}
}

func TestLoadPeerConfigTakesEveryNamedSetting(t *testing.T) {
	cfg, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvSeedlistURLs:           "https://seeds.example/seed.txt",
		yacypeerassembly.EnvDataDirectory:          "/var/lib/yacypeer",
		yacypeerassembly.EnvNetworkName:            "realm",
		yacypeerassembly.EnvOpsAddr:                "127.0.0.1:9099",
		yacypeerassembly.EnvEgressProxyURL:         "http://proxy.example:3128",
		yacypeerassembly.EnvMetricHistoryRetention: "48h",
	}))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	want := yacypeerassembly.PeerConfig{
		EgressProxyURL:         &url.URL{Scheme: "http", Host: "proxy.example:3128"},
		DataDirectory:          "/var/lib/yacypeer",
		NetworkName:            "realm",
		SeedlistURLs:           "https://seeds.example/seed.txt",
		OpsAddr:                "127.0.0.1:9099",
		MetricHistoryRetention: 48 * time.Hour,
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("config = %+v, want %+v", cfg, want)
	}
}

func TestLoadPeerConfigRefusesAnEgressProxyThatIsNotAnHTTPURL(t *testing.T) {
	_, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvSeedlistURLs:   "https://seeds.example/seed.txt",
		yacypeerassembly.EnvEgressProxyURL: "socks5://proxy.example:1080",
	}))
	if err == nil {
		t.Error("config loaded with a socks5 egress proxy, want an error")
	}
}

func TestLoadPeerConfigTakesTheYggdrasilPeering(t *testing.T) {
	cfg, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvYggdrasilPeers:  "tls://peer.example:443, quic://peer.example:443",
		yacypeerassembly.EnvYggdrasilListen: "tls://[::]:9001",
	}))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	want := yggdrasilnode.Peering{
		PeerURLs: []*url.URL{
			{Scheme: "tls", Host: "peer.example:443"},
			{Scheme: "quic", Host: "peer.example:443"},
		},
		ListenURLs: []*url.URL{{Scheme: "tls", Host: "[::]:9001"}},
	}
	if !reflect.DeepEqual(cfg.YggdrasilPeering, want) {
		t.Errorf("Yggdrasil peering = %+v, want %+v", cfg.YggdrasilPeering, want)
	}
}

func TestLoadPeerConfigRefusesAYggdrasilPeerWithoutAScheme(t *testing.T) {
	_, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvYggdrasilPeers: "peer.example",
	}))
	if err == nil {
		t.Error("config loaded with a Yggdrasil peer without a scheme, want an error")
	}
}

func TestLoadPeerConfigRefusesAYggdrasilListenURLThatDoesNotParse(t *testing.T) {
	_, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvYggdrasilListen: "tls://[::",
	}))
	if err == nil {
		t.Error("config loaded with a Yggdrasil listen URL that does not parse, want an error")
	}
}

func TestLoadPeerConfigTakesTheYggdrasilPeerList(t *testing.T) {
	cfg, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvYggdrasilPeerList: "https://peers.example/publicnodes.json",
	}))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	want := &url.URL{Scheme: "https", Host: "peers.example", Path: "/publicnodes.json"}
	if !reflect.DeepEqual(cfg.YggdrasilPeerList, want) {
		t.Errorf("Yggdrasil peer list = %v, want %v", cfg.YggdrasilPeerList, want)
	}
}

func TestLoadPeerConfigRefusesAYggdrasilPeerListBesideTheYggdrasilPeers(t *testing.T) {
	_, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvYggdrasilPeers:    "tls://peer.example:443",
		yacypeerassembly.EnvYggdrasilPeerList: "https://peers.example/publicnodes.json",
	}))
	if err == nil {
		t.Error("config loaded with a Yggdrasil peer list beside Yggdrasil peers, want an error")
	}
}

func TestLoadPeerConfigRefusesAYggdrasilPeerListThatIsNotAnHTTPURL(t *testing.T) {
	_, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvYggdrasilPeerList: "file:///etc/publicnodes.json",
	}))
	if err == nil {
		t.Error("config loaded with a file Yggdrasil peer list, want an error")
	}
}

func TestLoadPeerConfigRefusesAnEgressProxyBesideYggdrasil(t *testing.T) {
	_, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvYggdrasilPeers: "tls://peer.example:443",
		yacypeerassembly.EnvEgressProxyURL: "http://proxy.example:3128",
	}))
	if err == nil {
		t.Error("config loaded with an egress proxy beside Yggdrasil, want an error")
	}
}

func TestLoadPeerConfigRefusesARetentionThatIsNotADuration(t *testing.T) {
	_, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvMetricHistoryRetention: "a week",
	}))
	if err == nil {
		t.Fatal("config loaded a retention that is not a duration, want an error")
	}
}

func TestConfigureLoggingRefusesAnUnknownLevel(t *testing.T) {
	if err := yacypeerassembly.ConfigureLogging(envFrom(map[string]string{
		"LOG_LEVEL": "loud",
	})); err == nil {
		t.Error("logging configured with an unknown level, want an error")
	}
	if err := yacypeerassembly.ConfigureLogging(envFrom(nil)); err != nil {
		t.Errorf("ConfigureLogging: %v", err)
	}
}
