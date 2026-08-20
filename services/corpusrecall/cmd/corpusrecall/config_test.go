package main_test

import (
	"testing"
	"time"

	corpusrecall "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/cmd/corpusrecall"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadServiceConfigRequiresCrawlNATSURL(t *testing.T) {
	if _, err := corpusrecall.LoadServiceConfig(envFrom(map[string]string{
		corpusrecall.EnvPageMarkdownNATSURL: "nats://corpus:4222",
	})); err == nil {
		t.Fatal("expected error when CRAWL_NATS_URL is unset")
	}
}

func TestLoadServiceConfigRequiresPageMarkdownNATSURL(t *testing.T) {
	if _, err := corpusrecall.LoadServiceConfig(envFrom(map[string]string{
		corpusrecall.EnvCrawlNATSURL: "nats://crawl:4222",
	})); err == nil {
		t.Fatal("expected error when PAGE_MARKDOWN_NATS_URL is unset")
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := corpusrecall.LoadServiceConfig(envFrom(map[string]string{
		corpusrecall.EnvCrawlNATSURL:        "nats://crawl:4222",
		corpusrecall.EnvPageMarkdownNATSURL: "nats://corpus:4222",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.OrdersSubject != corpusrecall.DefaultOrdersSubject {
		t.Errorf("orders subject = %q", cfg.OrdersSubject)
	}
	if cfg.ListenAddr != corpusrecall.DefaultListenAddr ||
		cfg.OpsAddr != corpusrecall.DefaultOpsAddr {
		t.Errorf("addrs = %q %q", cfg.ListenAddr, cfg.OpsAddr)
	}
	if cfg.RecallLimit != corpusrecall.DefaultRecallLimit ||
		cfg.PollInterval != corpusrecall.DefaultPollInterval {
		t.Errorf("timings = %v %v", cfg.RecallLimit, cfg.PollInterval)
	}
	if cfg.MaxInFlight != corpusrecall.DefaultMaxInFlight ||
		cfg.MaxResponseBytes != corpusrecall.DefaultMaxResponseBytes {
		t.Errorf("limits = %d %d", cfg.MaxInFlight, cfg.MaxResponseBytes)
	}
	if cfg.CrawlNATSURL != "nats://crawl:4222" {
		t.Errorf("crawl nats url = %q", cfg.CrawlNATSURL)
	}
	if cfg.PageMarkdownNATSURL != "nats://corpus:4222" {
		t.Errorf("page markdown nats url = %q", cfg.PageMarkdownNATSURL)
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	cfg, err := corpusrecall.LoadServiceConfig(envFrom(map[string]string{
		corpusrecall.EnvCrawlNATSURL:        "nats://localhost:4222",
		corpusrecall.EnvPageMarkdownNATSURL: "nats://localhost:4222",
		corpusrecall.EnvOrdersSubject:       "t.orders",
		corpusrecall.EnvListenAddr:          "127.0.0.1:1000",
		corpusrecall.EnvOpsAddr:             "127.0.0.1:1001",
		corpusrecall.EnvRecallLimit:         "5s",
		corpusrecall.EnvPollInterval:        "250ms",
		corpusrecall.EnvMaxInFlight:         "8",
		corpusrecall.EnvMaxResponseBytes:    "1024",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.OrdersSubject != "t.orders" || cfg.ListenAddr != "127.0.0.1:1000" ||
		cfg.OpsAddr != "127.0.0.1:1001" {
		t.Errorf("strings = %+v", cfg)
	}
	if cfg.RecallLimit != 5*time.Second || cfg.PollInterval != 250*time.Millisecond {
		t.Errorf("timings = %v %v", cfg.RecallLimit, cfg.PollInterval)
	}
	if cfg.MaxInFlight != 8 || cfg.MaxResponseBytes != 1024 {
		t.Errorf("limits = %d %d", cfg.MaxInFlight, cfg.MaxResponseBytes)
	}
}

func TestLoadServiceConfigRejectsInvalidRecallLimit(t *testing.T) {
	if _, err := corpusrecall.LoadServiceConfig(envFrom(map[string]string{
		corpusrecall.EnvCrawlNATSURL:        "nats://localhost:4222",
		corpusrecall.EnvPageMarkdownNATSURL: "nats://localhost:4222",
		corpusrecall.EnvRecallLimit:         "soon",
	})); err == nil {
		t.Fatal("expected error for non-duration recall limit")
	}
}

func TestLoadServiceConfigRejectsInvalidMaxInFlight(t *testing.T) {
	if _, err := corpusrecall.LoadServiceConfig(envFrom(map[string]string{
		corpusrecall.EnvCrawlNATSURL:        "nats://localhost:4222",
		corpusrecall.EnvPageMarkdownNATSURL: "nats://localhost:4222",
		corpusrecall.EnvMaxInFlight:         "-1",
	})); err == nil {
		t.Fatal("expected error for non-positive max in flight")
	}
}
