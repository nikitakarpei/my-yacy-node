package main_test

import (
	"testing"

	corpustext "github.com/nikitakarpei/yacy-rwi-node/corpustext/cmd/corpustext"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadServiceConfigRequiresCrawlNATSURL(t *testing.T) {
	if _, err := corpustext.LoadServiceConfig(envFrom(nil)); err == nil {
		t.Fatal("expected error when CRAWL_NATS_URL is unset")
	}
}

func TestLoadServiceConfigRequiresSearchIndexEngine(t *testing.T) {
	if _, err := corpustext.LoadServiceConfig(envFrom(map[string]string{
		corpustext.EnvCrawlNATSURL: "nats://localhost:4222",
	})); err == nil {
		t.Fatal("expected error when SEARCH_INDEX_ENGINE is unset")
	}
}

func TestLoadServiceConfigRejectsUnknownSearchIndexEngine(t *testing.T) {
	if _, err := corpustext.LoadServiceConfig(envFrom(map[string]string{
		corpustext.EnvCrawlNATSURL:      "nats://localhost:4222",
		corpustext.EnvSearchIndexEngine: "sphinx",
	})); err == nil {
		t.Fatal("expected error for an unknown search index engine")
	}
}

func TestLoadServiceConfigRequiresElasticsearchURL(t *testing.T) {
	if _, err := corpustext.LoadServiceConfig(envFrom(map[string]string{
		corpustext.EnvCrawlNATSURL:      "nats://localhost:4222",
		corpustext.EnvSearchIndexEngine: corpustext.SearchIndexEngineElasticsearch,
	})); err == nil {
		t.Fatal("expected error when ELASTICSEARCH_URL is unset")
	}
}

func TestLoadServiceConfigManticoreRequiresURL(t *testing.T) {
	if _, err := corpustext.LoadServiceConfig(envFrom(map[string]string{
		corpustext.EnvCrawlNATSURL:      "nats://localhost:4222",
		corpustext.EnvSearchIndexEngine: corpustext.SearchIndexEngineManticore,
	})); err == nil {
		t.Fatal("expected error when MANTICORE_URL is unset")
	}
}

func TestLoadServiceConfigManticoreDefaults(t *testing.T) {
	cfg, err := corpustext.LoadServiceConfig(envFrom(map[string]string{
		corpustext.EnvCrawlNATSURL:      "nats://localhost:4222",
		corpustext.EnvSearchIndexEngine: corpustext.SearchIndexEngineManticore,
		corpustext.EnvManticoreURL:      "http://localhost:9308",
	}))
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
	cfg, err := corpustext.LoadServiceConfig(envFrom(map[string]string{
		corpustext.EnvCrawlNATSURL:      "nats://localhost:4222",
		corpustext.EnvSearchIndexEngine: corpustext.SearchIndexEngineElasticsearch,
		corpustext.EnvElasticsearchURL:  "http://localhost:9200",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.CrawledPageSubject != corpustext.DefaultCrawledPageSubject {
		t.Errorf("subject = %q", cfg.CrawledPageSubject)
	}
	if cfg.CrawledPageDurable != corpustext.DefaultCrawledPageDurable {
		t.Errorf("durable = %q", cfg.CrawledPageDurable)
	}
	if cfg.Concurrency != corpustext.DefaultConcurrency {
		t.Errorf("concurrency = %d", cfg.Concurrency)
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
	cfg, err := corpustext.LoadServiceConfig(envFrom(map[string]string{
		corpustext.EnvCrawlNATSURL:           "nats://localhost:4222",
		corpustext.EnvSearchIndexEngine:      corpustext.SearchIndexEngineElasticsearch,
		corpustext.EnvElasticsearchURL:       "http://localhost:9200",
		corpustext.EnvNATSCrawledPageSubject: "t.subject",
		corpustext.EnvNATSCrawledPageDurable: "dur",
		corpustext.EnvConcurrency:            "3",
		corpustext.EnvElasticsearchIndex:     "my_index",
		corpustext.EnvLanguages:              "en, de",
		corpustext.EnvOpsAddr:                "127.0.0.1:9099",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.CrawledPageSubject != "t.subject" {
		t.Errorf("subject = %q", cfg.CrawledPageSubject)
	}
	if cfg.CrawledPageDurable != "dur" || cfg.Concurrency != 3 {
		t.Errorf("durable/concurrency = %q %d", cfg.CrawledPageDurable, cfg.Concurrency)
	}
	if cfg.ElasticsearchIndex != "my_index" {
		t.Errorf("index = %q", cfg.ElasticsearchIndex)
	}
	if len(cfg.Languages) != 2 || cfg.Languages[0] != "en" || cfg.Languages[1] != "de" {
		t.Errorf("languages = %v", cfg.Languages)
	}
	if cfg.OpsAddr != "127.0.0.1:9099" {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
}
