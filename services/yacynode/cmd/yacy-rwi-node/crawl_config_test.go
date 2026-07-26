package main

import "testing"

func TestLoadCrawlConfigDisabledWhenNoURL(t *testing.T) {
	cfg := loadCrawlConfig(func(string) string { return "" })
	if cfg.Enabled() {
		t.Fatal("crawl should be disabled without NATS_URL")
	}
}

func TestLoadCrawlConfigDefaults(t *testing.T) {
	env := map[string]string{envNATSURL: "nats://localhost:4222"}
	cfg := loadCrawlConfig(func(k string) string { return env[k] })
	if !cfg.Enabled() {
		t.Fatal("crawl should be enabled")
	}
	if cfg.IngestSubject != defaultIngestSubject ||
		cfg.IngestDurable != defaultIngestDurable {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadCrawlConfigOverrides(t *testing.T) {
	env := map[string]string{
		envNATSURL:           "nats://localhost:4222",
		envNATSIngestSubject: "i",
		envNATSIngestDurable: "d",
	}
	cfg := loadCrawlConfig(func(k string) string { return env[k] })
	if cfg.IngestSubject != "i" || cfg.IngestDurable != "d" {
		t.Fatalf("overrides not applied: %+v", cfg)
	}
}
