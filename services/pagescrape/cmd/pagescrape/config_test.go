package main_test

import (
	"testing"
	"time"

	httppkg "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	pagescrape "github.com/nikitakarpei/yacy-rwi-node/pagescrape/cmd/pagescrape"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func requiredEnv() map[string]string {
	return map[string]string{
		pagescrape.EnvScrapeNATSURL: "nats://crawl:4222",
		pagescrape.EnvProxyURL:      "http://egress:3128",
	}
}

func TestLoadServiceConfigRequiresTheProxyURL(t *testing.T) {
	env := requiredEnv()
	delete(env, pagescrape.EnvProxyURL)
	if _, err := pagescrape.LoadServiceConfig(envFrom(env)); err == nil {
		t.Errorf("expected error when %s is unset", pagescrape.EnvProxyURL)
	}
}

func TestLoadServiceConfigLeavesTheNATSURLUnsetToServeItsOwnBroker(t *testing.T) {
	env := requiredEnv()
	delete(env, pagescrape.EnvScrapeNATSURL)
	cfg, err := pagescrape.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ScrapeNATSURL != "" {
		t.Errorf("nats url = %q", cfg.ScrapeNATSURL)
	}
	if cfg.NATSServer.Addr != pagescrape.DefaultNATSAddr ||
		cfg.NATSServer.StoreDir != pagescrape.DefaultNATSStoreDir {
		t.Errorf("nats server = %+v", cfg.NATSServer)
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := pagescrape.LoadServiceConfig(envFrom(requiredEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ScrapeRequestDurable != pagescrape.DefaultScrapeRequestDurable {
		t.Errorf("durable = %q", cfg.ScrapeRequestDurable)
	}
	if cfg.ProxyDialMode != httppkg.ProxyDialTunnel {
		t.Errorf("proxy dial mode = %d", cfg.ProxyDialMode)
	}
	if cfg.UserAgent != pagescrape.DefaultUserAgent {
		t.Errorf("user agent = %q", cfg.UserAgent)
	}
	if cfg.MaxBodyBytes != pagescrape.DefaultScrapeMaxBodyBytes ||
		cfg.FetchDeadline != pagescrape.DefaultScrapeFetchDeadline {
		t.Errorf("fetch limits = %d bytes, %s", cfg.MaxBodyBytes, cfg.FetchDeadline)
	}
	if cfg.ScrapeIntakeConcurrency != pagescrape.DefaultScrapeIntakeConcurrency {
		t.Errorf("intake concurrency = %d", cfg.ScrapeIntakeConcurrency)
	}
	if cfg.ScrapeDeferralWindow != pagescrape.DefaultScrapeDeferralWindow {
		t.Errorf("deferral window = %s", cfg.ScrapeDeferralWindow)
	}
	if cfg.OpsAddr != pagescrape.DefaultOpsAddr {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	env := requiredEnv()
	env[pagescrape.EnvScrapeRequestDurable] = "dur"
	env[pagescrape.EnvProxyDialMode] = "absolute-url"
	env[pagescrape.EnvUserAgent] = "agent (+https://example.test)"
	env[pagescrape.EnvScrapeMaxBodyBytes] = "4096"
	env[pagescrape.EnvScrapeFetchDeadline] = "5s"
	env[pagescrape.EnvScrapeIntakeConcurrency] = "3"
	env[pagescrape.EnvScrapeRequestsKept] = "9"
	env[pagescrape.EnvScrapeDeferralWindow] = "2h"
	env[pagescrape.EnvScrapePageOfferMaxBytes] = "8MB"
	env[pagescrape.EnvOpsAddr] = "127.0.0.1:9099"
	env[pagescrape.EnvNATSAddr] = ":4333"
	env[pagescrape.EnvNATSStoreDir] = "/data"

	cfg, err := pagescrape.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ScrapeRequestDurable != "dur" || cfg.ProxyDialMode != httppkg.ProxyDialAbsoluteURL {
		t.Errorf("durable/dial mode = %q %d", cfg.ScrapeRequestDurable, cfg.ProxyDialMode)
	}
	if cfg.UserAgent != "agent (+https://example.test)" {
		t.Errorf("user agent = %q", cfg.UserAgent)
	}
	if cfg.MaxBodyBytes != 4096 || cfg.FetchDeadline != 5*time.Second {
		t.Errorf("fetch limits = %d bytes, %s", cfg.MaxBodyBytes, cfg.FetchDeadline)
	}
	if cfg.ScrapeIntakeConcurrency != 3 || cfg.ScrapeRequestsKept != 9 {
		t.Errorf(
			"intake concurrency/kept = %d %d",
			cfg.ScrapeIntakeConcurrency,
			cfg.ScrapeRequestsKept,
		)
	}
	if cfg.ScrapeDeferralWindow != 2*time.Hour || cfg.PageOfferMaxBytes != 8<<20 {
		t.Errorf(
			"deferral window/offer max bytes = %s %d",
			cfg.ScrapeDeferralWindow,
			cfg.PageOfferMaxBytes,
		)
	}
	if cfg.OpsAddr != "127.0.0.1:9099" {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
	if cfg.NATSServer.Addr != ":4333" || cfg.NATSServer.StoreDir != "/data" {
		t.Errorf("nats server = %+v", cfg.NATSServer)
	}
}

func TestLoadServiceConfigRejectsWhatItCannotRead(t *testing.T) {
	rejected := map[string]map[string]string{
		"intake concurrency": {pagescrape.EnvScrapeIntakeConcurrency: "abc"},
		"requests kept":      {pagescrape.EnvScrapeRequestsKept: "-1"},
		"max body bytes":     {pagescrape.EnvScrapeMaxBodyBytes: "-1"},
		"fetch deadline":     {pagescrape.EnvScrapeFetchDeadline: "soon"},
		"deferral window":    {pagescrape.EnvScrapeDeferralWindow: "eventually"},
		"offer max bytes":    {pagescrape.EnvScrapePageOfferMaxBytes: "a lot"},
		"proxy dial mode":    {pagescrape.EnvProxyDialMode: "carrier-pigeon"},
		"proxy url":          {pagescrape.EnvProxyURL: "ftp://egress:3128"},
	}
	for name, overrides := range rejected {
		env := requiredEnv()
		for key, value := range overrides {
			env[key] = value
		}
		if _, err := pagescrape.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}
