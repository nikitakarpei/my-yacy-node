package main_test

import (
	"testing"

	corpusmarkdown "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/cmd/corpusmarkdown"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadServiceConfigRequiresNATSURL(t *testing.T) {
	if _, err := corpusmarkdown.LoadServiceConfig(envFrom(nil)); err == nil {
		t.Fatal("expected error when NATS_URL is unset")
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := corpusmarkdown.LoadServiceConfig(envFrom(map[string]string{
		corpusmarkdown.EnvNATSURL: "nats://localhost:4222",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.CrawledPageSubject != corpusmarkdown.DefaultCrawledPageSubject {
		t.Errorf("subject = %q", cfg.CrawledPageSubject)
	}
	if cfg.CrawledPageDurable != corpusmarkdown.DefaultCrawledPageDurable {
		t.Errorf("durable = %q", cfg.CrawledPageDurable)
	}
	if cfg.Concurrency != corpusmarkdown.DefaultConcurrency {
		t.Errorf("concurrency = %d", cfg.Concurrency)
	}
	if cfg.OpsAddr != corpusmarkdown.DefaultOpsAddr {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	cfg, err := corpusmarkdown.LoadServiceConfig(envFrom(map[string]string{
		corpusmarkdown.EnvNATSURL:                "nats://localhost:4222",
		corpusmarkdown.EnvNATSCrawledPageSubject: "t.subject",
		corpusmarkdown.EnvNATSCrawledPageDurable: "dur",
		corpusmarkdown.EnvConcurrency:            "3",
		corpusmarkdown.EnvOpsAddr:                "127.0.0.1:9099",
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
	if _, err := corpusmarkdown.LoadServiceConfig(envFrom(map[string]string{
		corpusmarkdown.EnvNATSURL:     "nats://localhost:4222",
		corpusmarkdown.EnvConcurrency: "abc",
	})); err == nil {
		t.Fatal("expected error for non-numeric concurrency")
	}
}
