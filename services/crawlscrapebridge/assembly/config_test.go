package crawlscrapebridgeassembly_test

import (
	"testing"

	crawlscrapebridge "github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/assembly"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func requiredEnv() map[string]string {
	return map[string]string{
		crawlscrapebridge.EnvCrawlNATSURL:         "nats://crawl:4222",
		crawlscrapebridge.EnvScrapeRequestNATSURL: "nats://scrape:4222",
	}
}

func TestLoadServiceConfigRequiresEveryBroker(t *testing.T) {
	for _, missing := range []string{
		crawlscrapebridge.EnvCrawlNATSURL,
		crawlscrapebridge.EnvScrapeRequestNATSURL,
	} {
		env := requiredEnv()
		delete(env, missing)
		if _, err := crawlscrapebridge.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("config loaded while %s is unset, want an error", missing)
		}
	}
}

func TestLoadServiceConfigFallsBackToTheDefaultOfEverySetting(t *testing.T) {
	cfg, err := crawlscrapebridge.LoadServiceConfig(envFrom(requiredEnv()))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	want := crawlscrapebridge.ServiceConfig{
		CrawlNATSURL:                "nats://crawl:4222",
		ScrapeRequestNATSURL:        "nats://scrape:4222",
		CrawledPageDurable:          crawlscrapebridge.DefaultCrawledPageDurable,
		CrawledPageRelayConcurrency: crawlscrapebridge.DefaultCrawledPageRelayConcurrency,
		OpsAddr:                     crawlscrapebridge.DefaultOpsAddr,
	}
	if cfg != want {
		t.Errorf("config = %+v, want %+v", cfg, want)
	}
}

func TestLoadServiceConfigTakesEveryNamedSetting(t *testing.T) {
	env := requiredEnv()
	env[crawlscrapebridge.EnvCrawledPageDurable] = "bridge"
	env[crawlscrapebridge.EnvCrawledPageRelayConcurrency] = "9"
	env[crawlscrapebridge.EnvOpsAddr] = "127.0.0.1:9099"

	cfg, err := crawlscrapebridge.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	if cfg.CrawledPageDurable != "bridge" || cfg.CrawledPageRelayConcurrency != 9 ||
		cfg.OpsAddr != "127.0.0.1:9099" {
		t.Errorf("config = %+v", cfg)
	}
}

func TestLoadServiceConfigRefusesAnUnreadableConcurrency(t *testing.T) {
	env := requiredEnv()
	env[crawlscrapebridge.EnvCrawledPageRelayConcurrency] = "many"
	if _, err := crawlscrapebridge.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("config loaded with an unreadable concurrency, want an error")
	}
}
