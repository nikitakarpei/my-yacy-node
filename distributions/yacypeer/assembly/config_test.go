package yacypeerassembly_test

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	yacypeerassembly "github.com/nikitakarpei/yacy-rwi-node/yacypeer/assembly"
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
		WebAddr:                yacypeerassembly.DefaultWebAddr,
		MetricHistoryRetention: yacypeerassembly.DefaultMetricHistoryRetention,
		PeerKeySearchTime:      yacypeerassembly.DefaultPeerKeySearchTime,
	}
	if cfg != want {
		t.Errorf("config = %+v, want %+v", cfg, want)
	}
}

func TestLoadPeerConfigTakesEveryNamedSetting(t *testing.T) {
	cfg, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvSeedlistURLs:           "https://seeds.example/seed.txt",
		yacypeerassembly.EnvDataDirectory:          "/var/lib/yacypeer",
		yacypeerassembly.EnvNetworkName:            "realm",
		yacypeerassembly.EnvWebAddr:                "127.0.0.1:9099",
		yacypeerassembly.EnvPeerProxyURL:           "http://proxy.example:3128",
		yacypeerassembly.EnvPageFetchProxyURL:      "http://127.0.0.1:8118",
		yacypeerassembly.EnvMetricHistoryRetention: "48h",
		yacypeerassembly.EnvPeerKeySearchTime:      "5m",
	}))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	want := yacypeerassembly.PeerConfig{
		PeerProxyURL:           &url.URL{Scheme: "http", Host: "proxy.example:3128"},
		PageFetchProxyURL:      &url.URL{Scheme: "http", Host: "127.0.0.1:8118"},
		DataDirectory:          "/var/lib/yacypeer",
		NetworkName:            "realm",
		SeedlistURLs:           "https://seeds.example/seed.txt",
		WebAddr:                "127.0.0.1:9099",
		MetricHistoryRetention: 48 * time.Hour,
		PeerKeySearchTime:      5 * time.Minute,
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("config = %+v, want %+v", cfg, want)
	}
}

func TestLoadPeerConfigRefusesAnEgressProxyThatIsNotAnHTTPURL(t *testing.T) {
	_, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvSeedlistURLs: "https://seeds.example/seed.txt",
		yacypeerassembly.EnvPeerProxyURL: "socks5://proxy.example:1080",
	}))
	if err == nil {
		t.Error("config loaded with a socks5 egress proxy, want an error")
	}
}

func TestLoadPeerConfigRefusesAPageFetchProxyItCannotSpeakTo(t *testing.T) {
	for _, proxyURL := range []string{"socks5h://127.0.0.1:9050", "proxy.example:3128", "://"} {
		_, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
			yacypeerassembly.EnvSeedlistURLs:      "https://seeds.example/seed.txt",
			yacypeerassembly.EnvPageFetchProxyURL: proxyURL,
		}))
		if err == nil {
			t.Errorf("config loaded with page fetch proxy %q, want an error", proxyURL)
		}
	}
}

func TestLoadPeerConfigSendsPageFetchesApartFromThePeerProxy(t *testing.T) {
	cfg, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
		yacypeerassembly.EnvSeedlistURLs: "https://seeds.example/seed.txt",
		yacypeerassembly.EnvPeerProxyURL: "http://proxy.example:3128",
	}))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	if cfg.PageFetchProxyURL != nil {
		t.Errorf("page fetch proxy = %v, want none", cfg.PageFetchProxyURL)
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

func TestLoadPeerConfigRefusesAProxyThatTheEnvironmentNames(t *testing.T) {
	for _, key := range []string{
		"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy", "NO_PROXY", "no_proxy",
	} {
		t.Run(key, func(t *testing.T) {
			_, err := yacypeerassembly.LoadPeerConfig(envFrom(map[string]string{
				key: "http://proxy.example:3128",
			}))
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Errorf("LoadPeerConfig = %v, want %s refused", err, key)
			}
		})
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
