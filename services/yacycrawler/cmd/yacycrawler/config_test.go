package main_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetchers/http"
	yacycrawler "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/cmd/yacycrawler"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func baseEnv() map[string]string {
	return map[string]string{
		"CRAWL_NATS_URL":        "nats://localhost:4222",
		"YACYCRAWLER_PROXY_URL": "http://proxy:8080",
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := yacycrawler.LoadServiceConfig(envFrom(baseEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.OrdersSubject != yacycrawler.DefaultOrdersSubject {
		t.Fatalf("orders subject = %q", cfg.OrdersSubject)
	}
	if cfg.FetchConcurrency != yacycrawler.DefaultFetchConcurrency ||
		cfg.RunPageBudget != yacycrawler.DefaultRunPageBudget {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.FetchDeadline != yacycrawler.DefaultFetchDeadline {
		t.Fatalf("fetch deadline = %v", cfg.FetchDeadline)
	}
	if cfg.ContentTypes != nil {
		t.Fatalf("content types should default empty, got %v", cfg.ContentTypes)
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
	env["YACYCRAWLER_PROXY_DIAL_MODE"] = "absolute-url"
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
	env["YACYCRAWLER_PROXY_DIAL_MODE"] = "nonsense"
	if _, err := yacycrawler.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("unknown proxy dial mode should error")
	}
}

func TestLoadServiceConfigOverridesUserAgent(t *testing.T) {
	env := baseEnv()
	env["YACYCRAWLER_USER_AGENT"] = "acme-crawler (+https://acme.test)"
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

func TestLoadServiceConfigRequiresProxy(t *testing.T) {
	env := baseEnv()
	delete(env, "YACYCRAWLER_PROXY_URL")
	if _, err := yacycrawler.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("missing proxy should error")
	}
}

func TestLoadServiceConfigRejectsNonHTTPProxy(t *testing.T) {
	env := baseEnv()
	env["YACYCRAWLER_PROXY_URL"] = "ftp://proxy"
	if _, err := yacycrawler.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("non-http proxy should error")
	}
}

func TestLoadServiceConfigParsesContentTypes(t *testing.T) {
	env := baseEnv()
	env["YACYCRAWLER_CONTENT_TYPES"] = "text/html, Application/PDF ,"
	cfg, err := yacycrawler.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.ContentTypes) != 2 ||
		cfg.ContentTypes[0] != "text/html" || cfg.ContentTypes[1] != "application/pdf" {
		t.Fatalf("content types = %v", cfg.ContentTypes)
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

func TestLoadServiceConfigRejectsBadValues(t *testing.T) {
	for _, bad := range []map[string]string{
		{"YACYCRAWLER_FETCH_CONCURRENCY": "0"},
		{"YACYCRAWLER_FETCH_CONCURRENCY": "notint"},
		{"YACYCRAWLER_MAX_BODY_BYTES": "-1"},
		{"YACYCRAWLER_FETCH_DEADLINE": "nope"},
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
