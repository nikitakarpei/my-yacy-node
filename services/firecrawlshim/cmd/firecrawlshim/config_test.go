package main_test

import (
	"testing"

	firecrawlshim "github.com/nikitakarpei/yacy-rwi-node/firecrawlshim/cmd/firecrawlshim"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func requiredEnv() map[string]string {
	return map[string]string{
		firecrawlshim.EnvCrawlNATSURL:         "nats://crawl:4222",
		firecrawlshim.EnvCrawlOutcomesTarget:  "crawler:8095",
		firecrawlshim.EnvMarkdownCorpusTarget: "corpusmarkdown:8094",
	}
}

func TestLoadServiceConfigRequiresEachCollaborator(t *testing.T) {
	for _, missing := range []string{
		firecrawlshim.EnvCrawlNATSURL,
		firecrawlshim.EnvCrawlOutcomesTarget,
		firecrawlshim.EnvMarkdownCorpusTarget,
	} {
		values := requiredEnv()
		delete(values, missing)
		if _, err := firecrawlshim.LoadServiceConfig(envFrom(values)); err == nil {
			t.Errorf("expected an error when %s is unset", missing)
		}
	}
}

func TestLoadServiceConfigRejectsAnInvalidRecallLimit(t *testing.T) {
	values := requiredEnv()
	values[firecrawlshim.EnvRecallLimit] = "nope"

	if _, err := firecrawlshim.LoadServiceConfig(envFrom(values)); err == nil {
		t.Fatal("expected an error for an invalid recall limit")
	}
}

func TestLoadServiceConfigRejectsAnInvalidPollInterval(t *testing.T) {
	values := requiredEnv()
	values[firecrawlshim.EnvPollInterval] = "nope"

	if _, err := firecrawlshim.LoadServiceConfig(envFrom(values)); err == nil {
		t.Fatal("expected an error for an invalid poll interval")
	}
}

func TestLoadServiceConfigRejectsAnInvalidInFlightLimit(t *testing.T) {
	values := requiredEnv()
	values[firecrawlshim.EnvMaxInFlight] = "0"

	if _, err := firecrawlshim.LoadServiceConfig(envFrom(values)); err == nil {
		t.Fatal("expected an error for an invalid in-flight limit")
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := firecrawlshim.LoadServiceConfig(envFrom(requiredEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if cfg.ListenAddr != firecrawlshim.DefaultListenAddr {
		t.Errorf("listen addr = %q", cfg.ListenAddr)
	}
	if cfg.OrdersSubject != firecrawlshim.DefaultOrdersSubject {
		t.Errorf("orders subject = %q", cfg.OrdersSubject)
	}
	if cfg.RecallLimit != firecrawlshim.DefaultRecallLimit {
		t.Errorf("recall limit = %s", cfg.RecallLimit)
	}
	if cfg.PollInterval != firecrawlshim.DefaultPollInterval {
		t.Errorf("poll interval = %s", cfg.PollInterval)
	}
	if cfg.MaxInFlight != firecrawlshim.DefaultMaxInFlight {
		t.Errorf("max in flight = %d", cfg.MaxInFlight)
	}
	if cfg.CrawlOutcomesTarget != "crawler:8095" {
		t.Errorf("crawl outcomes target = %q", cfg.CrawlOutcomesTarget)
	}
	if cfg.MarkdownCorpusTarget != "corpusmarkdown:8094" {
		t.Errorf("markdown corpus target = %q", cfg.MarkdownCorpusTarget)
	}
}
