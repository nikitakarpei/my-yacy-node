package main_test

import (
	"testing"
	"time"

	webresearchmcp "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/cmd/webresearchmcp"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func requiredEnv() map[string]string {
	return map[string]string{
		webresearchmcp.EnvSearXNGURL:           "http://searxng:8080",
		webresearchmcp.EnvScrapeRequestNATSURL: "nats://crawl:4222",
		webresearchmcp.EnvCorpusMarkdownAddr:   "corpusmarkdown:8094",
	}
}

func TestLoadServiceConfigRequiresEveryDependency(t *testing.T) {
	for _, missing := range []string{
		webresearchmcp.EnvSearXNGURL,
		webresearchmcp.EnvScrapeRequestNATSURL,
		webresearchmcp.EnvCorpusMarkdownAddr,
	} {
		env := requiredEnv()
		delete(env, missing)
		if _, err := webresearchmcp.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("config loaded while %s is unset, want an error", missing)
		}
	}
}

func TestLoadServiceConfigFallsBackToTheDefaultOfEverySetting(t *testing.T) {
	cfg, err := webresearchmcp.LoadServiceConfig(envFrom(requiredEnv()))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	if cfg.SearXNGSearchDeadline != webresearchmcp.DefaultSearXNGSearchDeadline {
		t.Errorf("searxng search deadline = %s", cfg.SearXNGSearchDeadline)
	}
	if cfg.PageFetchWait != webresearchmcp.DefaultPageFetchWait {
		t.Errorf("page fetch wait = %s", cfg.PageFetchWait)
	}
	if cfg.PageScrapeTolerance != webresearchmcp.DefaultPageScrapeTolerance {
		t.Errorf("page scrape tolerance = %s", cfg.PageScrapeTolerance)
	}
	if cfg.CorpusMarkdownRecallDeadline != webresearchmcp.DefaultCorpusMarkdownRecallDeadline {
		t.Errorf("corpus recall deadline = %s", cfg.CorpusMarkdownRecallDeadline)
	}
	if cfg.PageFetchCharacterLimit != webresearchmcp.DefaultPageFetchCharacterLimit {
		t.Errorf("page fetch character limit = %d", cfg.PageFetchCharacterLimit)
	}
	if cfg.SearchResultLimit != webresearchmcp.DefaultSearchResultLimit {
		t.Errorf("search result limit = %d", cfg.SearchResultLimit)
	}
	if cfg.ToolCallConcurrency != webresearchmcp.DefaultToolCallConcurrency {
		t.Errorf("tool call concurrency = %d", cfg.ToolCallConcurrency)
	}
	if cfg.ListenAddr != webresearchmcp.DefaultListenAddr {
		t.Errorf("listen addr = %q", cfg.ListenAddr)
	}
	if cfg.OpsAddr != webresearchmcp.DefaultOpsAddr {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
}

func TestLoadServiceConfigTakesEverySettingAnOperatorNames(t *testing.T) {
	env := requiredEnv()
	env[webresearchmcp.EnvSearXNGSearchDeadline] = "3s"
	env[webresearchmcp.EnvPageFetchWait] = "45s"
	env[webresearchmcp.EnvPageScrapeTolerance] = "15m"
	env[webresearchmcp.EnvCorpusMarkdownRecallDeadline] = "2s"
	env[webresearchmcp.EnvPageFetchCharacterLimit] = "120"
	env[webresearchmcp.EnvSearchResultLimit] = "4"
	env[webresearchmcp.EnvToolCallConcurrency] = "16"
	env[webresearchmcp.EnvListenAddr] = "127.0.0.1:9095"
	env[webresearchmcp.EnvOpsAddr] = "127.0.0.1:9099"

	cfg, err := webresearchmcp.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	if cfg.SearXNGSearchDeadline != 3*time.Second {
		t.Errorf("searxng search deadline = %s, want 3s", cfg.SearXNGSearchDeadline)
	}
	if cfg.PageFetchWait != 45*time.Second {
		t.Errorf("page fetch wait = %s, want 45s", cfg.PageFetchWait)
	}
	if cfg.PageScrapeTolerance != 15*time.Minute {
		t.Errorf("page scrape tolerance = %s, want 15m", cfg.PageScrapeTolerance)
	}
	if cfg.CorpusMarkdownRecallDeadline != 2*time.Second {
		t.Errorf("corpus recall deadline = %s, want 2s", cfg.CorpusMarkdownRecallDeadline)
	}
	if cfg.PageFetchCharacterLimit != 120 {
		t.Errorf("page fetch character limit = %d, want 120", cfg.PageFetchCharacterLimit)
	}
	if cfg.SearchResultLimit != 4 {
		t.Errorf("search result limit = %d, want 4", cfg.SearchResultLimit)
	}
	if cfg.ToolCallConcurrency != 16 {
		t.Errorf("tool call concurrency = %d, want 16", cfg.ToolCallConcurrency)
	}
	if cfg.ListenAddr != "127.0.0.1:9095" {
		t.Errorf("listen addr = %q", cfg.ListenAddr)
	}
	if cfg.OpsAddr != "127.0.0.1:9099" {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
}

func TestLoadServiceConfigRefusesASettingItCannotRead(t *testing.T) {
	for _, unreadable := range []struct {
		key   string
		value string
	}{
		{webresearchmcp.EnvSearXNGURL, "not a url"},
		{webresearchmcp.EnvSearXNGSearchDeadline, "soon"},
		{webresearchmcp.EnvPageFetchWait, "soon"},
		{webresearchmcp.EnvPageScrapeTolerance, "a while"},
		{webresearchmcp.EnvCorpusMarkdownRecallDeadline, "soon"},
		{webresearchmcp.EnvPageFetchCharacterLimit, "many"},
		{webresearchmcp.EnvSearchResultLimit, "0"},
		{webresearchmcp.EnvToolCallConcurrency, "-1"},
	} {
		env := requiredEnv()
		env[unreadable.key] = unreadable.value
		if _, err := webresearchmcp.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("config loaded with %s=%q, want an error", unreadable.key, unreadable.value)
		}
	}
}
