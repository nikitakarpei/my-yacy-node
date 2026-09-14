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

func TestLoadServiceConfigRequiresEveryAddress(t *testing.T) {
	for _, missing := range []string{pagescrape.EnvScrapeNATSURL, pagescrape.EnvProxyURL} {
		env := requiredEnv()
		delete(env, missing)
		if _, err := pagescrape.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("expected error when %s is unset", missing)
		}
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
	if cfg.ScrapeIntakeConcurrency != pagescrape.DefaultScrapeIntakeConcurrency ||
		cfg.ScrapeRequestsInFlight != pagescrape.DefaultScrapeRequestsInFlight {
		t.Errorf(
			"intake concurrency/in flight = %d %d",
			cfg.ScrapeIntakeConcurrency,
			cfg.ScrapeRequestsInFlight,
		)
	}
	if cfg.ScrapeDeferralWindow != pagescrape.DefaultScrapeDeferralWindow ||
		cfg.PageOfferMaxAge != pagescrape.DefaultScrapePageOfferMaxAge {
		t.Errorf(
			"deferral window/offer max age = %s %s",
			cfg.ScrapeDeferralWindow,
			cfg.PageOfferMaxAge,
		)
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
	env[pagescrape.EnvScrapeRequestsInFlight] = "7"
	env[pagescrape.EnvScrapeRequestsKept] = "9"
	env[pagescrape.EnvScrapeDeferralWindow] = "2h"
	env[pagescrape.EnvScrapePageOfferMaxBytes] = "8MB"
	env[pagescrape.EnvScrapePageOfferMaxAge] = "30m"
	env[pagescrape.EnvOpsAddr] = "127.0.0.1:9099"

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
	if cfg.ScrapeIntakeConcurrency != 3 || cfg.ScrapeRequestsInFlight != 7 ||
		cfg.ScrapeRequestsKept != 9 {
		t.Errorf(
			"intake concurrency/in flight/kept = %d %d %d",
			cfg.ScrapeIntakeConcurrency,
			cfg.ScrapeRequestsInFlight,
			cfg.ScrapeRequestsKept,
		)
	}
	if cfg.ScrapeDeferralWindow != 2*time.Hour || cfg.PageOfferMaxBytes != 8<<20 ||
		cfg.PageOfferMaxAge != 30*time.Minute {
		t.Errorf(
			"deferral window/offer limits = %s %d %s",
			cfg.ScrapeDeferralWindow,
			cfg.PageOfferMaxBytes,
			cfg.PageOfferMaxAge,
		)
	}
	if cfg.OpsAddr != "127.0.0.1:9099" {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
}

func TestLoadServiceConfigRejectsWhatItCannotRead(t *testing.T) {
	rejected := map[string]map[string]string{
		"intake concurrency": {pagescrape.EnvScrapeIntakeConcurrency: "abc"},
		"requests in flight": {pagescrape.EnvScrapeRequestsInFlight: "0"},
		"requests kept":      {pagescrape.EnvScrapeRequestsKept: "-1"},
		"max body bytes":     {pagescrape.EnvScrapeMaxBodyBytes: "-1"},
		"fetch deadline":     {pagescrape.EnvScrapeFetchDeadline: "soon"},
		"deferral window":    {pagescrape.EnvScrapeDeferralWindow: "eventually"},
		"offer max bytes":    {pagescrape.EnvScrapePageOfferMaxBytes: "a lot"},
		"offer max age":      {pagescrape.EnvScrapePageOfferMaxAge: "a while"},
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
