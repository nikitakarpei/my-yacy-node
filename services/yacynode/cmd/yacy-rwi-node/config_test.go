package main_test

import (
	"strings"
	"testing"
	"time"

	yacynode "github.com/nikitakarpei/yacy-rwi-node/yacynode/cmd/yacy-rwi-node"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadNodeConfigAppliesDefaults(t *testing.T) {
	config, err := yacynode.LoadNodeConfig(envFrom(map[string]string{
		yacynode.EnvPeerHash: "0123456789AB",
		yacynode.EnvPeerName: "node",
		yacynode.EnvProxyURL: "http://proxy:4750",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if config.ProxyURL == nil || config.ProxyURL.String() != "http://proxy:4750" {
		t.Errorf("ProxyURL = %v", config.ProxyURL)
	}
	if config.PeerAddr != yacynode.DefaultPeerAddr {
		t.Errorf("PeerAddr = %q, want %q", config.PeerAddr, yacynode.DefaultPeerAddr)
	}
	if config.OpsAddr != yacynode.DefaultOpsAddr {
		t.Errorf("OpsAddr = %q, want %q", config.OpsAddr, yacynode.DefaultOpsAddr)
	}
	if config.AdvertisePort != 8090 {
		t.Errorf("AdvertisePort = %d, want 8090 (from peer addr)", config.AdvertisePort)
	}
	if !strings.HasSuffix(config.StoragePath, yacynode.StorageFileName) {
		t.Errorf("StoragePath = %q, want suffix %q", config.StoragePath, yacynode.StorageFileName)
	}
	if config.StorageQuotaByte != 1<<30 {
		t.Errorf("StorageQuotaByte = %d, want 1GB", config.StorageQuotaByte)
	}
	if config.AnnounceInterval != yacynode.DefaultAnnounceInterval {
		t.Errorf("AnnounceInterval = %v, want default", config.AnnounceInterval)
	}
	if config.SeedlistURLs != nil {
		t.Errorf("SeedlistURLs = %v, want nil", config.SeedlistURLs)
	}
	if config.Crawl.Enabled() {
		t.Errorf("Crawl = %+v, want disabled without a broker", config.Crawl)
	}
}

func TestLoadNodeConfigDefaultsTheCrawlSubjectAndDurable(t *testing.T) {
	config, err := yacynode.LoadNodeConfig(envFrom(map[string]string{
		yacynode.EnvPeerHash: "0123456789AB",
		yacynode.EnvPeerName: "node",
		yacynode.EnvProxyURL: "http://proxy:4750",
		yacynode.EnvNATSURL:  "nats://localhost:4222",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if !config.Crawl.Enabled() {
		t.Fatalf("Crawl = %+v, want enabled by the broker url", config.Crawl)
	}
	if config.Crawl.IngestSubject != yacynode.DefaultIngestSubject {
		t.Errorf(
			"IngestSubject = %q, want %q",
			config.Crawl.IngestSubject,
			yacynode.DefaultIngestSubject,
		)
	}
	if config.Crawl.IngestDurable != yacynode.DefaultIngestDurable {
		t.Errorf(
			"IngestDurable = %q, want %q",
			config.Crawl.IngestDurable,
			yacynode.DefaultIngestDurable,
		)
	}
}

func TestLoadNodeConfigReadsOverrides(t *testing.T) {
	config, err := yacynode.LoadNodeConfig(envFrom(map[string]string{
		yacynode.EnvPeerHash:          "0123456789AB",
		yacynode.EnvPeerName:          "node",
		yacynode.EnvProxyURL:          "http://proxy:4750",
		yacynode.EnvNetworkName:       "testnet",
		yacynode.EnvPeerAddr:          ":7000",
		yacynode.EnvOpsAddr:           ":7001",
		yacynode.EnvAdvertiseHost:     "203.0.113.1",
		yacynode.EnvAdvertisePort:     "9999",
		yacynode.EnvStorageQuota:      "2MB",
		yacynode.EnvTrustedProxies:    "10.0.0.0/8",
		yacynode.EnvSeedlistURLs:      " http://a , http://b ,",
		yacynode.EnvAnnounceInterval:  "30s",
		yacynode.EnvNATSURL:           "nats://broker:4222",
		yacynode.EnvNATSIngestSubject: "ingest.subject",
		yacynode.EnvNATSIngestDurable: "ingest-durable",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if config.NetworkName != "testnet" {
		t.Errorf("NetworkName = %q", config.NetworkName)
	}
	if config.AdvertisePort != 9999 {
		t.Errorf("AdvertisePort = %d, want 9999", config.AdvertisePort)
	}
	if config.StorageQuotaByte != 2<<20 {
		t.Errorf("StorageQuotaByte = %d, want 2MB", config.StorageQuotaByte)
	}
	if len(config.TrustedProxyNetworks) != 1 {
		t.Errorf("TrustedProxyNetworks = %d, want 1", len(config.TrustedProxyNetworks))
	}
	if got := config.SeedlistURLs; len(got) != 2 || got[0] != "http://a" || got[1] != "http://b" {
		t.Errorf("SeedlistURLs = %v, want trimmed pair", got)
	}
	if config.AnnounceInterval != 30*time.Second {
		t.Errorf("AnnounceInterval = %v, want 30s", config.AnnounceInterval)
	}
	if config.Crawl.IngestSubject != "ingest.subject" ||
		config.Crawl.IngestDurable != "ingest-durable" {
		t.Errorf("Crawl = %+v, want the named subject and durable", config.Crawl)
	}
}

func TestLoadNodeConfigReadsEveryTrustedProxyNotation(t *testing.T) {
	config, err := yacynode.LoadNodeConfig(envFrom(map[string]string{
		yacynode.EnvPeerHash:       "0123456789AB",
		yacynode.EnvPeerName:       "node",
		yacynode.EnvProxyURL:       "http://proxy:4750",
		yacynode.EnvTrustedProxies: "10.0.0.1, 192.168.0.0/16, , ::1",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(config.TrustedProxyNetworks) != 3 {
		t.Fatalf(
			"TrustedProxyNetworks = %v, want a network per non-empty entry",
			config.TrustedProxyNetworks,
		)
	}
}

func TestLoadNodeConfigRejects(t *testing.T) {
	cases := map[string]map[string]string{
		"bad hash":     {yacynode.EnvPeerHash: "short"},
		"missing name": {yacynode.EnvPeerHash: "0123456789AB"},
		"announce no host": {
			yacynode.EnvPeerHash:     "0123456789AB",
			yacynode.EnvPeerName:     "n",
			yacynode.EnvSeedlistURLs: "http://seed",
		},
		"bad port": {
			yacynode.EnvPeerHash:      "0123456789AB",
			yacynode.EnvPeerName:      "n",
			yacynode.EnvAdvertisePort: "-3",
		},
		"bad quota": {
			yacynode.EnvPeerHash:     "0123456789AB",
			yacynode.EnvPeerName:     "n",
			yacynode.EnvStorageQuota: "big",
		},
		"bad announce interval": {
			yacynode.EnvPeerHash:         "0123456789AB",
			yacynode.EnvPeerName:         "n",
			yacynode.EnvAnnounceInterval: "nope",
		},
		"negative announce interval": {
			yacynode.EnvPeerHash:         "0123456789AB",
			yacynode.EnvPeerName:         "n",
			yacynode.EnvAnnounceInterval: "-1s",
		},
		"bad trusted proxy ip": {
			yacynode.EnvPeerHash:       "0123456789AB",
			yacynode.EnvPeerName:       "n",
			yacynode.EnvTrustedProxies: "999.0.0.1",
		},
		"bad trusted proxy mask": {
			yacynode.EnvPeerHash:       "0123456789AB",
			yacynode.EnvPeerName:       "n",
			yacynode.EnvTrustedProxies: "10.0.0.0/99",
		},
		"missing proxy url": {
			yacynode.EnvPeerHash: "0123456789AB",
			yacynode.EnvPeerName: "n",
		},
		"non-http proxy url": {
			yacynode.EnvPeerHash: "0123456789AB",
			yacynode.EnvPeerName: "n",
			yacynode.EnvProxyURL: "socks5://proxy:1080",
		},
	}
	for name, env := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := yacynode.LoadNodeConfig(envFrom(env)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
