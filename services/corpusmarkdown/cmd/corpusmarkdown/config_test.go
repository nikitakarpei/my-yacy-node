package main_test

import (
	"testing"
	"time"

	corpusmarkdown "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/cmd/corpusmarkdown"
	httppkg "github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetchers/http"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func requiredEnv() map[string]string {
	return map[string]string{
		corpusmarkdown.EnvCrawlNATSURL:        "nats://crawl:4222",
		corpusmarkdown.EnvPageMarkdownNATSURL: "nats://corpus:4222",
		corpusmarkdown.EnvProxyURL:            "http://egress:3128",
	}
}

func TestLoadServiceConfigRequiresEveryAddress(t *testing.T) {
	for _, missing := range []string{
		corpusmarkdown.EnvCrawlNATSURL,
		corpusmarkdown.EnvPageMarkdownNATSURL,
		corpusmarkdown.EnvProxyURL,
	} {
		env := requiredEnv()
		delete(env, missing)
		if _, err := corpusmarkdown.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("expected error when %s is unset", missing)
		}
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := corpusmarkdown.LoadServiceConfig(envFrom(requiredEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ScrapeRequestSubject != corpusmarkdown.DefaultScrapeRequestSubject {
		t.Errorf("subject = %q", cfg.ScrapeRequestSubject)
	}
	if cfg.ScrapeRequestDurable != corpusmarkdown.DefaultScrapeRequestDurable {
		t.Errorf("durable = %q", cfg.ScrapeRequestDurable)
	}
	if cfg.Concurrency != corpusmarkdown.DefaultConcurrency {
		t.Errorf("concurrency = %d", cfg.Concurrency)
	}
	if cfg.OpsAddr != corpusmarkdown.DefaultOpsAddr {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
	if cfg.ProxyDialMode != httppkg.ProxyDialTunnel {
		t.Errorf("proxy dial mode = %d", cfg.ProxyDialMode)
	}
	if cfg.MaxBodyBytes != corpusmarkdown.DefaultMaxBodyBytes ||
		cfg.FetchDeadline != corpusmarkdown.DefaultFetchDeadline {
		t.Errorf("fetch limits = %d bytes, %s", cfg.MaxBodyBytes, cfg.FetchDeadline)
	}
	if cfg.UserAgent != corpusmarkdown.DefaultUserAgent {
		t.Errorf("user agent = %q", cfg.UserAgent)
	}
	if cfg.ProxyURL.Host != "egress:3128" {
		t.Errorf("proxy url = %q", cfg.ProxyURL)
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	env := requiredEnv()
	env[corpusmarkdown.EnvNATSScrapeRequestSubject] = "t.subject"
	env[corpusmarkdown.EnvNATSScrapeRequestDurable] = "dur"
	env[corpusmarkdown.EnvProxyDialMode] = "absolute-url"
	env[corpusmarkdown.EnvUserAgent] = "agent (+https://example.test)"
	env[corpusmarkdown.EnvMaxBodyBytes] = "4096"
	env[corpusmarkdown.EnvFetchDeadline] = "5s"
	env[corpusmarkdown.EnvConcurrency] = "3"
	env[corpusmarkdown.EnvOpsAddr] = "127.0.0.1:9099"

	cfg, err := corpusmarkdown.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ScrapeRequestSubject != "t.subject" || cfg.ScrapeRequestDurable != "dur" {
		t.Errorf("subject/durable = %q %q", cfg.ScrapeRequestSubject, cfg.ScrapeRequestDurable)
	}
	if cfg.ProxyDialMode != httppkg.ProxyDialAbsoluteURL {
		t.Errorf("proxy dial mode = %d", cfg.ProxyDialMode)
	}
	if cfg.UserAgent != "agent (+https://example.test)" {
		t.Errorf("user agent = %q", cfg.UserAgent)
	}
	if cfg.MaxBodyBytes != 4096 || cfg.FetchDeadline != 5*time.Second {
		t.Errorf("fetch limits = %d bytes, %s", cfg.MaxBodyBytes, cfg.FetchDeadline)
	}
	if cfg.Concurrency != 3 || cfg.OpsAddr != "127.0.0.1:9099" {
		t.Errorf("concurrency/ops addr = %d %q", cfg.Concurrency, cfg.OpsAddr)
	}
}

func TestLoadServiceConfigRejectsWhatItCannotRead(t *testing.T) {
	rejected := map[string]map[string]string{
		"concurrency":     {corpusmarkdown.EnvConcurrency: "abc"},
		"max body bytes":  {corpusmarkdown.EnvMaxBodyBytes: "-1"},
		"fetch deadline":  {corpusmarkdown.EnvFetchDeadline: "soon"},
		"proxy dial mode": {corpusmarkdown.EnvProxyDialMode: "carrier-pigeon"},
		"proxy url":       {corpusmarkdown.EnvProxyURL: "ftp://egress:3128"},
	}
	for name, overrides := range rejected {
		env := requiredEnv()
		for key, value := range overrides {
			env[key] = value
		}
		if _, err := corpusmarkdown.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}
