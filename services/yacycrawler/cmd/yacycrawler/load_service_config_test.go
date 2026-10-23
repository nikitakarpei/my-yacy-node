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

func TestLoadServiceConfigRequiresCrawlNATSURL(t *testing.T) {
	env := baseEnv()
	delete(env, "CRAWL_NATS_URL")
	if _, err := yacycrawler.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("missing CRAWL_NATS_URL should error")
	}
}

func TestLoadServiceConfigKeepsCrawlStateInTheDefaultDirectory(t *testing.T) {
	cfg, err := yacycrawler.LoadServiceConfig(envFrom(baseEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.CrawlStateDir != yacycrawler.DefaultCrawlStateDir {
		t.Fatalf(
			"crawl state directory = %q, want %q",
			cfg.CrawlStateDir,
			yacycrawler.DefaultCrawlStateDir,
		)
	}
	if cfg.CrawlStateLimits.BlockCacheBytes != 64<<20 ||
		cfg.CrawlStateLimits.WriteConcurrency != yacycrawler.DefaultCrawlStatePebbleWriteConcurrency {
		t.Fatalf("crawl state limits = %+v, want the defaults", cfg.CrawlStateLimits)
	}
}

func TestLoadServiceConfigReadsTheCrawlStateDirectoryAndLimits(t *testing.T) {
	env := baseEnv()
	env["CRAWL_STATE_DIR"] = "/data"
	env["CRAWL_STATE_PEBBLE_BLOCK_CACHE"] = "16MB"
	env["CRAWL_STATE_PEBBLE_MEMTABLE_SIZE"] = "4MB"
	env["CRAWL_STATE_PEBBLE_COMPACTION_CONCURRENCY"] = "2"
	env["CRAWL_STATE_PEBBLE_OPEN_FILE_LIMIT"] = "500"
	env["CRAWL_STATE_PEBBLE_WRITE_CONCURRENCY"] = "4"
	cfg, err := yacycrawler.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	limits := cfg.CrawlStateLimits
	if cfg.CrawlStateDir != "/data" || limits.BlockCacheBytes != 16<<20 ||
		limits.MemtableBytes != 4<<20 || limits.CompactionConcurrency != 2 ||
		limits.OpenFileLimit != 500 || limits.WriteConcurrency != 4 {
		t.Fatalf("crawl state = %q %+v, want the overrides", cfg.CrawlStateDir, limits)
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
	cfg, err := yacycrawler.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FetchConcurrency != 8 || cfg.FetchDeadline != 5*time.Second {
		t.Fatalf("overrides not applied: %+v", cfg)
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
		{"CRAWL_STATE_PEBBLE_BLOCK_CACHE": "lots"},
		{"CRAWL_STATE_PEBBLE_MEMTABLE_SIZE": "lots"},
		{"CRAWL_STATE_PEBBLE_COMPACTION_CONCURRENCY": "0"},
		{"CRAWL_STATE_PEBBLE_OPEN_FILE_LIMIT": "0"},
		{"CRAWL_STATE_PEBBLE_WRITE_CONCURRENCY": "0"},
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

func TestTheOperatorAPIServesOnlyAtAConfiguredAddress(t *testing.T) {
	off, err := yacycrawler.LoadServiceConfig(envFrom(baseEnv()))
	if err != nil {
		t.Fatal(err)
	}
	env := baseEnv()
	env["YACYCRAWLER_API_ADDR"] = ":9095"
	on, err := yacycrawler.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatal(err)
	}

	if off.ServesAPI() {
		t.Error("the operator API serves without an address")
	}
	if !on.ServesAPI() || on.APIAddr != ":9095" {
		t.Errorf("API address = %q, want :9095 served", on.APIAddr)
	}
}
