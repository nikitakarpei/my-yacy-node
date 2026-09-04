package main_test

import (
	"testing"
	"time"

	visitcrawl "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/cmd/visitcrawl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func baseEnv() map[string]string {
	return map[string]string{
		"CRAWL_NATS_URL":         "nats://localhost:4222",
		"VISITCRAWL_LINK_SECRET": "shared-secret",
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := visitcrawl.LoadServiceConfig(envFrom(baseEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.CrawlOrdersSubject != visitcrawl.DefaultCrawlOrdersSubject {
		t.Fatalf("orders subject = %q", cfg.CrawlOrdersSubject)
	}
	if cfg.ListenAddr != visitcrawl.DefaultListenAddr || cfg.OpsAddr != visitcrawl.DefaultOpsAddr {
		t.Fatalf("unexpected addr defaults: %+v", cfg)
	}
	if cfg.OrderTimeout != visitcrawl.DefaultOrderTimeout {
		t.Fatalf("order timeout = %v", cfg.OrderTimeout)
	}
	if cfg.MaxInFlight != visitcrawl.DefaultMaxInFlight {
		t.Fatalf("max in flight = %d", cfg.MaxInFlight)
	}
	if cfg.MaxBodyBytes != visitcrawl.DefaultMaxBodyBytes {
		t.Fatalf("max body bytes = %d", cfg.MaxBodyBytes)
	}
	if cfg.LinkSecret != "shared-secret" {
		t.Fatalf("link secret = %q", cfg.LinkSecret)
	}
	if cfg.CrawlProfile.Scope != yacycrawlcontract.ScopeDomain {
		t.Fatalf("scope = %v, want domain", cfg.CrawlProfile.Scope)
	}
	if cfg.CrawlProfile.URLMustMatch != yacycrawlcontract.MatchAll {
		t.Fatalf("urlMustMatch = %q, want MatchAll", cfg.CrawlProfile.URLMustMatch)
	}
	if cfg.CrawlProfile.MaxDepth != visitcrawl.DefaultCrawlMaxDepth {
		t.Fatalf("max depth = %d", cfg.CrawlProfile.MaxDepth)
	}
	if cfg.CrawlProfile.MaxPagesPerHost != visitcrawl.DefaultCrawlMaxPagesPerHost {
		t.Fatalf("max pages per host = %d", cfg.CrawlProfile.MaxPagesPerHost)
	}
}

func TestLoadServiceConfigRequiresCrawlNATSURL(t *testing.T) {
	env := baseEnv()
	delete(env, "CRAWL_NATS_URL")
	if _, err := visitcrawl.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("missing CRAWL_NATS_URL should error")
	}
}

func TestLoadServiceConfigRequiresLinkSecret(t *testing.T) {
	env := baseEnv()
	delete(env, visitcrawl.EnvLinkSecret)
	if _, err := visitcrawl.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatalf("missing %s should error", visitcrawl.EnvLinkSecret)
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	env := baseEnv()
	env["VISITCRAWL_LISTEN_ADDR"] = ":9000"
	env["VISITCRAWL_ORDER_TIMEOUT"] = "2s"
	env["VISITCRAWL_MAX_IN_FLIGHT"] = "8"
	env["VISITCRAWL_SCOPE"] = "wide"
	env["VISITCRAWL_MAX_DEPTH"] = "3"
	env["VISITCRAWL_ALLOW_QUERY_URLS"] = "true"
	cfg, err := visitcrawl.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != ":9000" || cfg.OrderTimeout != 2*time.Second || cfg.MaxInFlight != 8 {
		t.Fatalf("overrides not applied: %+v", cfg)
	}
	if cfg.CrawlProfile.Scope != yacycrawlcontract.ScopeWide || cfg.CrawlProfile.MaxDepth != 3 {
		t.Fatalf("crawl profile overrides not applied: %+v", cfg.CrawlProfile)
	}
	if !cfg.CrawlProfile.AllowQueryURLs {
		t.Fatal("allow query urls should be true")
	}
}

func TestLoadServiceConfigRejectsUnknownScope(t *testing.T) {
	env := baseEnv()
	env["VISITCRAWL_SCOPE"] = "galaxy"
	if _, err := visitcrawl.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("unknown scope should error")
	}
}

func TestLoadServiceConfigRejectsBadValues(t *testing.T) {
	for _, bad := range []map[string]string{
		{"VISITCRAWL_ORDER_TIMEOUT": "0s"},
		{"VISITCRAWL_ORDER_TIMEOUT": "nope"},
		{"VISITCRAWL_MAX_IN_FLIGHT": "0"},
		{"VISITCRAWL_MAX_BODY_BYTES": "-1"},
		{"VISITCRAWL_MAX_DEPTH": "-1"},
		{"VISITCRAWL_MAX_PAGES_PER_HOST": "0"},
		{"VISITCRAWL_ALLOW_QUERY_URLS": "maybe"},
		{"VISITCRAWL_URL_MUST_MATCH": "([unclosed"},
		{"VISITCRAWL_URL_MUST_NOT_MATCH": "([unclosed"},
	} {
		env := baseEnv()
		for k, v := range bad {
			env[k] = v
		}
		if _, err := visitcrawl.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("expected error for %v", bad)
		}
	}
}
