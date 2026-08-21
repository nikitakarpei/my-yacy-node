package main_test

import (
	"testing"
	"time"

	corpustext "github.com/nikitakarpei/yacy-rwi-node/corpustext/cmd/corpustext"
	httppkg "github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetchers/http"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func requiredEnv() map[string]string {
	return map[string]string{
		corpustext.EnvCrawlNATSURL:      "nats://crawl:4222",
		corpustext.EnvProxyURL:          "http://egress:3128",
		corpustext.EnvSearchIndexEngine: corpustext.SearchIndexEngineElasticsearch,
		corpustext.EnvElasticsearchURL:  "http://localhost:9200",
	}
}

func TestLoadServiceConfigRequiresEveryAddress(t *testing.T) {
	for _, missing := range []string{
		corpustext.EnvCrawlNATSURL,
		corpustext.EnvProxyURL,
		corpustext.EnvSearchIndexEngine,
		corpustext.EnvElasticsearchURL,
	} {
		env := requiredEnv()
		delete(env, missing)
		if _, err := corpustext.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("expected error when %s is unset", missing)
		}
	}
}

func TestLoadServiceConfigRejectsUnknownSearchIndexEngine(t *testing.T) {
	env := requiredEnv()
	env[corpustext.EnvSearchIndexEngine] = "sphinx"
	if _, err := corpustext.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("expected error for an unknown search index engine")
	}
}

func TestLoadServiceConfigManticoreRequiresURL(t *testing.T) {
	env := requiredEnv()
	env[corpustext.EnvSearchIndexEngine] = corpustext.SearchIndexEngineManticore
	if _, err := corpustext.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("expected error when MANTICORE_URL is unset")
	}
}

func TestLoadServiceConfigManticoreDefaults(t *testing.T) {
	env := requiredEnv()
	env[corpustext.EnvSearchIndexEngine] = corpustext.SearchIndexEngineManticore
	env[corpustext.EnvManticoreURL] = "http://localhost:9308"

	cfg, err := corpustext.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ManticoreURL != "http://localhost:9308" {
		t.Errorf("manticore url = %q", cfg.ManticoreURL)
	}
	if cfg.ManticoreTable != corpustext.DefaultIndexBaseName {
		t.Errorf("manticore table = %q", cfg.ManticoreTable)
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := corpustext.LoadServiceConfig(envFrom(requiredEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ReachedPageSubject != corpustext.DefaultReachedPageSubject {
		t.Errorf("subject = %q", cfg.ReachedPageSubject)
	}
	if cfg.ReachedPageDurable != corpustext.DefaultReachedPageDurable {
		t.Errorf("durable = %q", cfg.ReachedPageDurable)
	}
	if cfg.Concurrency != corpustext.DefaultConcurrency {
		t.Errorf("concurrency = %d", cfg.Concurrency)
	}
	if cfg.ProxyDialMode != httppkg.ProxyDialTunnel || cfg.ProxyURL.Host != "egress:3128" {
		t.Errorf("proxy = %q, dial mode %d", cfg.ProxyURL, cfg.ProxyDialMode)
	}
	if cfg.MaxBodyBytes != corpustext.DefaultMaxBodyBytes ||
		cfg.FetchDeadline != corpustext.DefaultFetchDeadline {
		t.Errorf("fetch limits = %d bytes, %s", cfg.MaxBodyBytes, cfg.FetchDeadline)
	}
	if cfg.UserAgent != corpustext.DefaultUserAgent {
		t.Errorf("user agent = %q", cfg.UserAgent)
	}
	if cfg.ElasticsearchIndex != corpustext.DefaultIndexBaseName {
		t.Errorf("index = %q", cfg.ElasticsearchIndex)
	}
	if cfg.OpsAddr != corpustext.DefaultOpsAddr {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
	if len(cfg.Languages) != 0 {
		t.Errorf("languages = %v", cfg.Languages)
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	env := requiredEnv()
	env[corpustext.EnvNATSReachedPageSubject] = "t.subject"
	env[corpustext.EnvNATSReachedPageDurable] = "dur"
	env[corpustext.EnvProxyDialMode] = "absolute-url"
	env[corpustext.EnvUserAgent] = "agent (+https://example.test)"
	env[corpustext.EnvMaxBodyBytes] = "4096"
	env[corpustext.EnvFetchDeadline] = "5s"
	env[corpustext.EnvConcurrency] = "3"
	env[corpustext.EnvElasticsearchIndex] = "my_index"
	env[corpustext.EnvLanguages] = "en, de"
	env[corpustext.EnvOpsAddr] = "127.0.0.1:9099"

	cfg, err := corpustext.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ReachedPageSubject != "t.subject" || cfg.ReachedPageDurable != "dur" {
		t.Errorf("subject/durable = %q %q", cfg.ReachedPageSubject, cfg.ReachedPageDurable)
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
	if cfg.ElasticsearchIndex != "my_index" {
		t.Errorf("index = %q", cfg.ElasticsearchIndex)
	}
	if len(cfg.Languages) != 2 || cfg.Languages[0] != "en" || cfg.Languages[1] != "de" {
		t.Errorf("languages = %v", cfg.Languages)
	}
}

func TestLoadServiceConfigRejectsWhatItCannotRead(t *testing.T) {
	rejected := map[string]map[string]string{
		"concurrency":     {corpustext.EnvConcurrency: "abc"},
		"max body bytes":  {corpustext.EnvMaxBodyBytes: "-1"},
		"fetch deadline":  {corpustext.EnvFetchDeadline: "soon"},
		"proxy dial mode": {corpustext.EnvProxyDialMode: "carrier-pigeon"},
		"proxy url":       {corpustext.EnvProxyURL: "ftp://egress:3128"},
	}
	for name, overrides := range rejected {
		env := requiredEnv()
		for key, value := range overrides {
			env[key] = value
		}
		if _, err := corpustext.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}
