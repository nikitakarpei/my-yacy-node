package main_test

import (
	"net/http"
	"strings"
	"testing"

	cdxscrape "github.com/nikitakarpei/yacy-rwi-node/cdxscrape/cmd/cdxscrape"
)

func TestLoadCommandConfigReadsTheStatedQuery(t *testing.T) {
	cfg, err := cdxscrape.LoadCommandConfig([]string{
		"-cdx-url", "http://pywb:8080",
		"-collection", "archive",
		"-url", "example.com",
		"-match-type", "host",
		"-from", "2024",
		"-to", "2025",
		"-limit", "50",
	}, environment(map[string]string{"SCRAPE_REQUEST_NATS_URL": "nats://nats:4222"}))
	if err != nil {
		t.Fatalf("load command config: %v", err)
	}

	if cfg.CDXURL.String() != "http://pywb:8080" {
		t.Errorf("cdx url = %q", cfg.CDXURL)
	}
	if cfg.Collection != "archive" {
		t.Errorf("collection = %q", cfg.Collection)
	}
	if cfg.Query.URL != "example.com" || cfg.Query.MatchType != "host" {
		t.Errorf("query = %+v", cfg.Query)
	}
	if cfg.Query.From != "2024" || cfg.Query.To != "2025" || cfg.Query.Limit != 50 {
		t.Errorf("query bounds = %+v", cfg.Query)
	}
	if cfg.ScrapeRequestNATSURL != "nats://nats:4222" {
		t.Errorf("nats url = %q", cfg.ScrapeRequestNATSURL)
	}
	if cfg.DryRun {
		t.Error("dry run = true, want false")
	}
}

func TestLoadCommandConfigReadsReplaysWhereItReadsTheIndexUnlessToldOtherwise(t *testing.T) {
	cfg := loadedConfig(t, "-cdx-url", "http://pywb:8080",
		"-collection", "archive", "-url", "example.com", "-dry-run")

	if cfg.ReplayURL.String() != "http://pywb:8080" {
		t.Errorf("replay url = %q, want the cdx url", cfg.ReplayURL)
	}
}

func TestLoadCommandConfigReadsReplaysWhereTheScraperReachesTheArchive(t *testing.T) {
	cfg := loadedConfig(t, "-cdx-url", "http://localhost:8080",
		"-replay-url", "http://pywb:8080",
		"-collection", "archive", "-url", "example.com", "-dry-run")

	if cfg.CDXURL.String() != "http://localhost:8080" {
		t.Errorf("cdx url = %q", cfg.CDXURL)
	}
	if cfg.ReplayURL.String() != "http://pywb:8080" {
		t.Errorf("replay url = %q", cfg.ReplayURL)
	}
}

func TestLoadCommandConfigAsksOnlyForPagesACorpusCanRead(t *testing.T) {
	cfg := loadedConfig(t, "-cdx-url", "http://pywb:8080",
		"-collection", "archive", "-url", "example.com", "-dry-run")

	if cfg.Query.MediaType != "text/html" {
		t.Errorf("media type = %q, want text/html", cfg.Query.MediaType)
	}
	if cfg.Query.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want %d", cfg.Query.StatusCode, http.StatusOK)
	}
}

func TestLoadCommandConfigSearchesAWholeDomainUnlessToldOtherwise(t *testing.T) {
	cfg := loadedConfig(t, "-cdx-url", "http://pywb:8080",
		"-collection", "archive", "-url", "example.com", "-dry-run")

	if cfg.Query.MatchType != "domain" {
		t.Errorf("match type = %q, want domain", cfg.Query.MatchType)
	}
}

func TestLoadCommandConfigNeedsNoBrokerForADryRun(t *testing.T) {
	cfg := loadedConfig(t, "-cdx-url", "http://pywb:8080",
		"-collection", "archive", "-url", "example.com", "-dry-run")

	if !cfg.DryRun {
		t.Error("dry run = false, want true")
	}
}

func TestLoadCommandConfigRefusesAnIncompleteCommand(t *testing.T) {
	for name, arguments := range map[string][]string{
		"no cdx url":    {"-collection", "archive", "-url", "example.com"},
		"no collection": {"-cdx-url", "http://pywb:8080", "-url", "example.com"},
		"no url":        {"-cdx-url", "http://pywb:8080", "-collection", "archive"},
		"cdx url without a scheme": {
			"-cdx-url", "pywb:8080", "-collection", "archive", "-url", "example.com",
		},
		"cdx url without a host": {
			"-cdx-url", "http://", "-collection", "archive", "-url", "example.com",
		},
		"unreadable cdx url": {
			"-cdx-url", "http://%zz", "-collection", "archive", "-url", "example.com",
		},
		"replay url without a host": {
			"-cdx-url", "http://pywb:8080", "-replay-url", "http://",
			"-collection", "archive", "-url", "example.com",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := cdxscrape.LoadCommandConfig(
				arguments,
				environment(map[string]string{"SCRAPE_REQUEST_NATS_URL": "nats://nats:4222"}),
			); err == nil {
				t.Fatal("load command config: want an error")
			}
		})
	}
}

func TestLoadCommandConfigRefusesToPublishWithoutABroker(t *testing.T) {
	_, err := cdxscrape.LoadCommandConfig([]string{
		"-cdx-url", "http://pywb:8080",
		"-collection", "archive",
		"-url", "example.com",
	}, environment(nil))
	if err == nil {
		t.Fatal("load command config: want an error")
	}
	if !strings.Contains(err.Error(), "SCRAPE_REQUEST_NATS_URL") {
		t.Fatalf("error = %v, want it to name the broker address", err)
	}
}

func TestLoadCommandConfigRefusesAnUnknownArgument(t *testing.T) {
	if _, err := cdxscrape.LoadCommandConfig(
		[]string{"-coll", "archive"},
		environment(nil),
	); err == nil {
		t.Fatal("load command config: want an error")
	}
}

func loadedConfig(t *testing.T, arguments ...string) cdxscrape.CommandConfig {
	t.Helper()
	cfg, err := cdxscrape.LoadCommandConfig(arguments, environment(nil))
	if err != nil {
		t.Fatalf("load command config: %v", err)
	}
	return cfg
}

func environment(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}
