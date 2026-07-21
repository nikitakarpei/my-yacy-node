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

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL: "nats://localhost:4222",
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
	if cfg.OpsAddr != DefaultOpsAddr {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	cfg, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL:                "nats://localhost:4222",
		EnvNATSCrawledPageSubject: "t.subject",
		EnvNATSCrawledPageDurable: "dur",
		EnvConcurrency:            "3",
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
	if cfg.OpsAddr != "127.0.0.1:9099" {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
}

func TestLoadServiceConfigRejectsInvalidConcurrency(t *testing.T) {
	if _, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL:     "nats://localhost:4222",
		EnvConcurrency: "abc",
	})); err == nil {
		t.Fatal("expected error for non-numeric concurrency")
	}
}
