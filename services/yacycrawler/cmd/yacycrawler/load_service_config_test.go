package main_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	yacycrawler "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/cmd/yacycrawler"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func baseEnv() map[string]string {
	return map[string]string{
		"CRAWL_NATS_URL":              "nats://localhost:4222",
		"CRAWL_STATE_REDIS_URL":       "redis://localhost:6379/0",
		"YACYCRAWLER_FETCH_PROXY_URL": "http://proxy:8080",
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := yacycrawler.LoadServiceConfig(envFrom(baseEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.CrawlOrdersSubject != yacycrawler.DefaultCrawlOrdersSubject {
		t.Fatalf("orders subject = %q", cfg.CrawlOrdersSubject)
	}
	if cfg.FetchConcurrency != yacycrawler.DefaultFetchConcurrency {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.FetchDeadline != yacycrawler.DefaultFetchDeadline {
		t.Fatalf("fetch deadline = %v", cfg.FetchDeadline)
	}
	if cfg.UserAgent != yacycrawler.DefaultUserAgent {
		t.Fatalf("user agent = %q", cfg.UserAgent)
	}
	if cfg.ProxyDialMode != http.ProxyDialTunnel {
		t.Fatalf("proxy dial mode = %v, want tunnel", cfg.ProxyDialMode)
	}
}

func TestLoadServiceConfigAcceptsAbsoluteURLDialMode(t *testing.T) {
	env := baseEnv()
	env["YACYCRAWLER_FETCH_PROXY_DIAL_MODE"] = "absolute-url"
	cfg, err := yacycrawler.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ProxyDialMode != http.ProxyDialAbsoluteURL {
		t.Fatalf("proxy dial mode = %v, want absolute-url", cfg.ProxyDialMode)
	}
}

func TestLoadServiceConfigRejectsUnknownDialMode(t *testing.T) {
	env := baseEnv()
	env["YACYCRAWLER_FETCH_PROXY_DIAL_MODE"] = "nonsense"
	if _, err := yacycrawler.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("unknown proxy dial mode should error")
	}
}

func TestLoadServiceConfigOverridesUserAgent(t *testing.T) {
	env := baseEnv()
	env["YACYCRAWLER_FETCH_USER_AGENT"] = "acme-crawler (+https://acme.test)"
	cfg, err := yacycrawler.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.UserAgent != "acme-crawler (+https://acme.test)" {
		t.Fatalf("user agent = %q", cfg.UserAgent)
	}
}

func TestLoadServiceConfigLeavesTheCrawlNATSURLUnsetToServeItsOwnBroker(t *testing.T) {
	env := baseEnv()
	delete(env, "CRAWL_NATS_URL")
	cfg, err := yacycrawler.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.CrawlNATSURL != "" {
		t.Fatalf("crawl nats url = %q", cfg.CrawlNATSURL)
	}
	if cfg.NATSServer.Addr != yacycrawler.DefaultNATSAddr ||
		cfg.NATSServer.StoreDir != yacycrawler.DefaultNATSStoreDir {
		t.Fatalf("nats server = %+v", cfg.NATSServer)
	}
}

func TestLoadServiceConfigReadsTheCrawlStateServer(t *testing.T) {
	cfg, err := yacycrawler.LoadServiceConfig(envFrom(baseEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.CrawlStateRedis.Addr != "localhost:6379" {
		t.Fatalf("Redis address = %q, want localhost:6379", cfg.CrawlStateRedis.Addr)
	}
}

func TestLoadServiceConfigRequiresCrawlStateRedisURL(t *testing.T) {
	env := baseEnv()
	delete(env, "CRAWL_STATE_REDIS_URL")
	if _, err := yacycrawler.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("missing CRAWL_STATE_REDIS_URL should error")
	}
}

func TestLoadServiceConfigRequiresProxy(t *testing.T) {
	env := baseEnv()
	delete(env, "YACYCRAWLER_FETCH_PROXY_URL")
	if _, err := yacycrawler.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("missing proxy should error")
	}
}

func TestLoadServiceConfigRejectsNonHTTPProxy(t *testing.T) {
	env := baseEnv()
	env["YACYCRAWLER_FETCH_PROXY_URL"] = "ftp://proxy"
	if _, err := yacycrawler.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("non-http proxy should error")
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	env := baseEnv()
	env["YACYCRAWLER_FETCH_CONCURRENCY"] = "8"
	env["YACYCRAWLER_FETCH_DEADLINE"] = "5s"
	env["YACYCRAWLER_NATS_ADDR"] = ":4333"
	env["YACYCRAWLER_NATS_STORE_DIR"] = "/data"
	cfg, err := yacycrawler.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FetchConcurrency != 8 || cfg.FetchDeadline != 5*time.Second {
		t.Fatalf("overrides not applied: %+v", cfg)
	}
	if cfg.NATSServer.Addr != ":4333" || cfg.NATSServer.StoreDir != "/data" {
		t.Fatalf("nats server = %+v", cfg.NATSServer)
	}
}

func TestPendingVisitLeaseOutlastsTheFetchDeadline(t *testing.T) {
	env := baseEnv()
	env["YACYCRAWLER_FETCH_DEADLINE"] = "5s"
	cfg, err := yacycrawler.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PendingPageVisitLease() <= 5*time.Second {
		t.Fatalf("pending page visit lease = %v, want longer than the fetch deadline",
			cfg.PendingPageVisitLease())
	}
}

func TestLoadServiceConfigRejectsBadValues(t *testing.T) {
	for _, bad := range []map[string]string{
		{"YACYCRAWLER_FETCH_CONCURRENCY": "0"},
		{"YACYCRAWLER_FETCH_CONCURRENCY": "notint"},
		{"YACYCRAWLER_MAX_BODY_BYTES": "-1"},
		{"YACYCRAWLER_FETCH_DEADLINE": "nope"},
		{"CRAWL_STATE_REDIS_URL": "http://localhost:6379"},
	} {
		env := baseEnv()
		for k, v := range bad {
			env[k] = v
		}
		if _, err := yacycrawler.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("expected error for %v", bad)
		}
	}
}
