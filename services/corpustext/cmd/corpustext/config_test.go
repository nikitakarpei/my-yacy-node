package main

import "testing"

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestStartReturnsNonZeroOnInvalidConfig(t *testing.T) {
	origLookup := lookupEnv
	lookupEnv = envFrom(nil)
	defer func() { lookupEnv = origLookup }()

	if code := start(); code != 2 {
		t.Errorf("start() = %d, want 2", code)
	}
}

func TestLoadServiceConfigRequiresNATSURL(t *testing.T) {
	if _, err := LoadServiceConfig(envFrom(nil)); err == nil {
		t.Fatal("expected error when NATS_URL is unset")
	}
}

func TestLoadServiceConfigRequiresSearchIndexEngine(t *testing.T) {
	if _, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL: "nats://localhost:4222",
	})); err == nil {
		t.Fatal("expected error when SEARCH_INDEX_ENGINE is unset")
	}
}

func TestLoadServiceConfigRequiresElasticsearchURL(t *testing.T) {
	if _, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL:           "nats://localhost:4222",
		EnvSearchIndexEngine: SearchIndexEngineElasticsearch,
	})); err == nil {
		t.Fatal("expected error when ELASTICSEARCH_URL is unset")
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL:           "nats://localhost:4222",
		EnvSearchIndexEngine: SearchIndexEngineElasticsearch,
		EnvElasticsearchURL:  "http://localhost:9200",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.CrawledPageSubject != DefaultCrawledPageSubject {
		t.Errorf("subject = %q", cfg.CrawledPageSubject)
	}
	if cfg.CrawledPageDurable != DefaultCrawledPageDurable {
		t.Errorf("durable = %q", cfg.CrawledPageDurable)
	}
	if cfg.Concurrency != DefaultConcurrency {
		t.Errorf("concurrency = %d", cfg.Concurrency)
	}
	if cfg.ElasticsearchIndex != DefaultIndexBaseName {
		t.Errorf("index = %q", cfg.ElasticsearchIndex)
	}
	if cfg.OpsAddr != DefaultOpsAddr {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
	if len(cfg.Languages) != 0 {
		t.Errorf("languages = %v", cfg.Languages)
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	cfg, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL:                "nats://localhost:4222",
		EnvSearchIndexEngine:      SearchIndexEngineElasticsearch,
		EnvElasticsearchURL:       "http://localhost:9200",
		EnvNATSCrawledPageSubject: "t.subject",
		EnvNATSCrawledPageDurable: "dur",
		EnvConcurrency:            "3",
		EnvElasticsearchIndex:     "my_index",
		EnvLanguages:              "en, de",
		EnvOpsAddr:                "127.0.0.1:9099",
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
