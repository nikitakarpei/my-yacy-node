package nodeconfiguration_test

import (
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadAppliesDefaults(t *testing.T) {
	config, err := nodeconfiguration.Load(envFrom(map[string]string{
		nodeconfiguration.EnvInitialPeerHash: "0123456789AB",
		nodeconfiguration.EnvPeerName:        "node",
		nodeconfiguration.EnvEgressProxyURL:  "http://proxy:4750",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if config.Egress.ProxyURL == nil || config.Egress.ProxyURL.String() != "http://proxy:4750" {
		t.Errorf("ProxyURL = %v", config.Egress.ProxyURL)
	}
	if config.Serving.PeerAddr != nodeconfiguration.DefaultPeerAddr {
		t.Errorf(
			"PeerAddr = %q, want %q",
			config.Serving.PeerAddr,
			nodeconfiguration.DefaultPeerAddr,
		)
	}
	if config.Serving.OpsAddr != nodeconfiguration.DefaultOpsAddr {
		t.Errorf("OpsAddr = %q, want %q", config.Serving.OpsAddr, nodeconfiguration.DefaultOpsAddr)
	}
	if config.Identity.AdvertisePort != 8090 {
		t.Errorf("AdvertisePort = %d, want 8090 (from peer addr)", config.Identity.AdvertisePort)
	}
	if !strings.HasSuffix(config.Storage.Path, nodeconfiguration.StorageFileName) {
		t.Errorf(
			"StoragePath = %q, want suffix %q",
			config.Storage.Path,
			nodeconfiguration.StorageFileName,
		)
	}
	if config.Storage.QuotaByte != 1<<30 {
		t.Errorf("StorageQuotaByte = %d, want 1GB", config.Storage.QuotaByte)
	}
	if config.PeerExchange.AnnounceInterval != nodeconfiguration.DefaultAnnounceInterval {
		t.Errorf("AnnounceInterval = %v, want default", config.PeerExchange.AnnounceInterval)
	}
	if config.PeerExchange.SeedlistURLs != nil {
		t.Errorf("SeedlistURLs = %v, want nil", config.PeerExchange.SeedlistURLs)
	}
	if config.ScrapeRequestIngest.Enabled() {
		t.Errorf(
			"ScrapeRequestIngest = %+v, want disabled without a broker",
			config.ScrapeRequestIngest,
		)
	}
}

func TestLoadDefaultsTheScrapeRequestIngest(t *testing.T) {
	config, err := nodeconfiguration.Load(envFrom(map[string]string{
		nodeconfiguration.EnvInitialPeerHash:      "0123456789AB",
		nodeconfiguration.EnvPeerName:             "node",
		nodeconfiguration.EnvEgressProxyURL:       "http://proxy:4750",
		nodeconfiguration.EnvScrapeRequestNATSURL: "nats://localhost:4222",
		nodeconfiguration.EnvScrapeProxyURL:       "http://renderproxy:8080",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if !config.ScrapeRequestIngest.Enabled() {
		t.Fatalf(
			"ScrapeRequestIngest = %+v, want enabled by the broker url",
			config.ScrapeRequestIngest,
		)
	}
	if config.ScrapeRequestIngest.ScrapeRequestSubject != nodeconfiguration.DefaultScrapeRequestSubject ||
		config.ScrapeRequestIngest.ScrapeRequestDurable != nodeconfiguration.DefaultScrapeRequestDurable {
		t.Errorf(
			"ScrapeRequestIngest = %+v, want the default subject and durable",
			config.ScrapeRequestIngest,
		)
	}
	if config.ScrapeRequestIngest.UserAgent != nodeconfiguration.DefaultScrapeUserAgent ||
		config.ScrapeRequestIngest.MaxBodyBytes != nodeconfiguration.DefaultScrapeMaxBodyBytes ||
		config.ScrapeRequestIngest.FetchDeadline != nodeconfiguration.DefaultScrapeFetchDeadline ||
		config.ScrapeRequestIngest.ScrapeRequestIntakeConcurrency != nodeconfiguration.DefaultScrapeRequestIntakeConcurrency {
		t.Errorf(
			"ScrapeRequestIngest = %+v, want the default fetch settings",
			config.ScrapeRequestIngest,
		)
	}
	if config.ScrapeRequestIngest.ProxyURL == nil ||
		config.ScrapeRequestIngest.ProxyURL.Host != "renderproxy:8080" {
		t.Errorf(
			"scrape request ingest proxy = %v, want the configured proxy",
			config.ScrapeRequestIngest.ProxyURL,
		)
	}
}

func TestLoadRequiresAScrapeProxyWhenIngestIsEnabled(t *testing.T) {
	_, err := nodeconfiguration.Load(envFrom(map[string]string{
		nodeconfiguration.EnvInitialPeerHash:      "0123456789AB",
		nodeconfiguration.EnvPeerName:             "node",
		nodeconfiguration.EnvEgressProxyURL:       "http://proxy:4750",
		nodeconfiguration.EnvScrapeRequestNATSURL: "nats://localhost:4222",
	}))

	if err == nil {
		t.Fatal("scrape request ingest without a fetch proxy should fail")
	}
}

func TestLoadReadsOverrides(t *testing.T) {
	config, err := nodeconfiguration.Load(envFrom(map[string]string{
		nodeconfiguration.EnvInitialPeerHash:                "0123456789AB",
		nodeconfiguration.EnvPeerName:                       "node",
		nodeconfiguration.EnvEgressProxyURL:                 "http://proxy:4750",
		nodeconfiguration.EnvNetworkName:                    "testnet",
		nodeconfiguration.EnvPeerAddr:                       ":7000",
		nodeconfiguration.EnvOpsAddr:                        ":7001",
		nodeconfiguration.EnvAdvertiseHost:                  "203.0.113.1",
		nodeconfiguration.EnvAdvertisePort:                  "9999",
		nodeconfiguration.EnvStorageQuota:                   "2MB",
		nodeconfiguration.EnvTrustedProxies:                 "10.0.0.0/8",
		nodeconfiguration.EnvSeedlistURLs:                   " http://a , http://b ,",
		nodeconfiguration.EnvAnnounceInterval:               "30s",
		nodeconfiguration.EnvScrapeRequestNATSURL:           "nats://broker:4222",
		nodeconfiguration.EnvScrapeProxyURL:                 "http://renderproxy:8080",
		nodeconfiguration.EnvScrapeRequestSubject:           "reached.subject",
		nodeconfiguration.EnvScrapeRequestDurable:           "reached-durable",
		nodeconfiguration.EnvScrapeRequestIntakeConcurrency: "9",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if config.Identity.NetworkName != "testnet" {
		t.Errorf("NetworkName = %q", config.Identity.NetworkName)
	}
	if config.Identity.AdvertisePort != 9999 {
		t.Errorf("AdvertisePort = %d, want 9999", config.Identity.AdvertisePort)
	}
	if config.Storage.QuotaByte != 2<<20 {
		t.Errorf("StorageQuotaByte = %d, want 2MB", config.Storage.QuotaByte)
	}
	if len(config.Serving.TrustedProxyNetworks) != 1 {
		t.Errorf("TrustedProxyNetworks = %d, want 1", len(config.Serving.TrustedProxyNetworks))
	}
	if got := config.PeerExchange.SeedlistURLs; len(got) != 2 || got[0] != "http://a" ||
		got[1] != "http://b" {
		t.Errorf("SeedlistURLs = %v, want trimmed pair", got)
	}
	if config.PeerExchange.AnnounceInterval != 30*time.Second {
		t.Errorf("AnnounceInterval = %v, want 30s", config.PeerExchange.AnnounceInterval)
	}
	if config.ScrapeRequestIngest.ScrapeRequestSubject != "reached.subject" ||
		config.ScrapeRequestIngest.ScrapeRequestDurable != "reached-durable" ||
		config.ScrapeRequestIngest.ScrapeRequestIntakeConcurrency != 9 {
		t.Errorf(
			"ScrapeRequestIngest = %+v, want the named subject, durable, and scrapeRequestIntakeConcurrency",
			config.ScrapeRequestIngest,
		)
	}
}

func TestLoadReadsEveryTrustedProxyNotation(t *testing.T) {
	config, err := nodeconfiguration.Load(envFrom(map[string]string{
		nodeconfiguration.EnvInitialPeerHash: "0123456789AB",
		nodeconfiguration.EnvPeerName:        "node",
		nodeconfiguration.EnvEgressProxyURL:  "http://proxy:4750",
		nodeconfiguration.EnvTrustedProxies:  "10.0.0.1, 192.168.0.0/16, , ::1",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(config.Serving.TrustedProxyNetworks) != 3 {
		t.Fatalf(
			"TrustedProxyNetworks = %v, want a network per non-empty entry",
			config.Serving.TrustedProxyNetworks,
		)
	}
}

func TestLoadLeavesTheInitialPeerHashUnstated(t *testing.T) {
	config, err := nodeconfiguration.Load(envFrom(map[string]string{
		nodeconfiguration.EnvPeerName:       "node",
		nodeconfiguration.EnvEgressProxyURL: "http://proxy:4750",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if config.Identity.InitialHash.Present() {
		t.Errorf("InitialHash = %v, want absent without the variable", config.Identity.InitialHash)
	}
}

func TestLoadLeavesThePeerNameUnstated(t *testing.T) {
	config, err := nodeconfiguration.Load(envFrom(map[string]string{
		nodeconfiguration.EnvEgressProxyURL: "http://proxy:4750",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if config.Identity.Name.Present() {
		t.Errorf("Name = %v, want absent without the variable", config.Identity.Name)
	}
}

func TestLoadRejects(t *testing.T) {
	cases := map[string]map[string]string{
		"bad hash": {nodeconfiguration.EnvInitialPeerHash: "short"},
		"bad name": {
			nodeconfiguration.EnvInitialPeerHash: "0123456789AB",
			nodeconfiguration.EnvPeerName:        "has space",
			nodeconfiguration.EnvEgressProxyURL:  "http://proxy:4750",
		},
		"announce no host": {
			nodeconfiguration.EnvInitialPeerHash: "0123456789AB",
			nodeconfiguration.EnvPeerName:        "n",
			nodeconfiguration.EnvSeedlistURLs:    "http://seed",
		},
		"bad port": {
			nodeconfiguration.EnvInitialPeerHash: "0123456789AB",
			nodeconfiguration.EnvPeerName:        "n",
			nodeconfiguration.EnvAdvertisePort:   "-3",
		},
		"bad quota": {
			nodeconfiguration.EnvInitialPeerHash: "0123456789AB",
			nodeconfiguration.EnvPeerName:        "n",
			nodeconfiguration.EnvStorageQuota:    "big",
		},
		"bad announce interval": {
			nodeconfiguration.EnvInitialPeerHash:  "0123456789AB",
			nodeconfiguration.EnvPeerName:         "n",
			nodeconfiguration.EnvAnnounceInterval: "nope",
		},
		"negative announce interval": {
			nodeconfiguration.EnvInitialPeerHash:  "0123456789AB",
			nodeconfiguration.EnvPeerName:         "n",
			nodeconfiguration.EnvAnnounceInterval: "-1s",
		},
		"bad trusted proxy ip": {
			nodeconfiguration.EnvInitialPeerHash: "0123456789AB",
			nodeconfiguration.EnvPeerName:        "n",
			nodeconfiguration.EnvTrustedProxies:  "999.0.0.1",
		},
		"bad trusted proxy mask": {
			nodeconfiguration.EnvInitialPeerHash: "0123456789AB",
			nodeconfiguration.EnvPeerName:        "n",
			nodeconfiguration.EnvTrustedProxies:  "10.0.0.0/99",
		},
		"missing proxy url": {
			nodeconfiguration.EnvInitialPeerHash: "0123456789AB",
			nodeconfiguration.EnvPeerName:        "n",
		},
		"non-http proxy url": {
			nodeconfiguration.EnvInitialPeerHash: "0123456789AB",
			nodeconfiguration.EnvPeerName:        "n",
			nodeconfiguration.EnvEgressProxyURL:  "socks5://proxy:1080",
		},
	}
	for name, env := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := nodeconfiguration.Load(envFrom(env)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
