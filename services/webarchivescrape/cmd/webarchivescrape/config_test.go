package main_test

import (
	"net/http"
	"strings"
	"testing"

	webarchivescrape "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/cmd/webarchivescrape"
)

func TestLoadCommandConfigReadsTheStatedQuery(t *testing.T) {
	cfg, err := webarchivescrape.LoadCommandConfig([]string{
		"-pywb-url", "http://pywb:8080",
		"-pywb-collection", "archive",
		"-url", "example.com",
		"-match-type", "host",
		"-from", "2024",
		"-to", "2025",
		"-limit", "50",
	}, environment(map[string]string{"SCRAPE_REQUEST_NATS_URL": "nats://nats:4222"}))
	if err != nil {
		t.Fatalf("load command config: %v", err)
	}

	if cfg.PywbURL.String() != "http://pywb:8080" {
		t.Errorf("pywb url = %q", cfg.PywbURL)
	}
	if cfg.PywbCollection != "archive" {
		t.Errorf("collection = %q", cfg.PywbCollection)
	}
	if cfg.PywbQuery.URL != "example.com" || cfg.PywbQuery.MatchType != "host" {
		t.Errorf("query = %+v", cfg.PywbQuery)
	}
	if cfg.PywbQuery.From != "2024" || cfg.PywbQuery.To != "2025" || cfg.PywbQuery.Limit != 50 {
		t.Errorf("query bounds = %+v", cfg.PywbQuery)
	}
	if cfg.ScrapeRequestNATSURL != "nats://nats:4222" {
		t.Errorf("nats url = %q", cfg.ScrapeRequestNATSURL)
	}
	if cfg.DryRun {
		t.Error("dry run = true, want false")
	}
}

func TestLoadCommandConfigAsksOnlyForPagesACorpusCanRead(t *testing.T) {
	cfg := loadedConfig(t, "-pywb-url", "http://pywb:8080",
		"-pywb-collection", "archive", "-url", "example.com", "-dry-run")

	if cfg.PywbQuery.MediaType != "text/html" {
		t.Errorf("media type = %q, want text/html", cfg.PywbQuery.MediaType)
	}
	if cfg.PywbQuery.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want %d", cfg.PywbQuery.StatusCode, http.StatusOK)
	}
}

func TestLoadCommandConfigSearchesAWholeDomainUnlessToldOtherwise(t *testing.T) {
	cfg := loadedConfig(t, "-pywb-url", "http://pywb:8080",
		"-pywb-collection", "archive", "-url", "example.com", "-dry-run")

	if cfg.PywbQuery.MatchType != "domain" {
		t.Errorf("match type = %q, want domain", cfg.PywbQuery.MatchType)
	}
}

func TestLoadCommandConfigNeedsNoBrokerForADryRun(t *testing.T) {
	cfg := loadedConfig(t, "-pywb-url", "http://pywb:8080",
		"-pywb-collection", "archive", "-url", "example.com", "-dry-run")

	if !cfg.DryRun {
		t.Error("dry run = false, want true")
	}
}

func TestLoadCommandConfigRefusesAnIncompleteCommand(t *testing.T) {
	for name, arguments := range map[string][]string{
		"no pywb url":        {"-pywb-collection", "archive", "-url", "example.com"},
		"no pywb collection": {"-pywb-url", "http://pywb:8080", "-url", "example.com"},
		"no url":             {"-pywb-url", "http://pywb:8080", "-pywb-collection", "archive"},
		"pywb url without a scheme": {
			"-pywb-url", "pywb:8080", "-pywb-collection", "archive", "-url", "example.com",
		},
		"pywb url without a host": {
			"-pywb-url", "http://", "-pywb-collection", "archive", "-url", "example.com",
		},
		"unreadable cdx url": {
			"-pywb-url", "http://%zz", "-pywb-collection", "archive", "-url", "example.com",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := webarchivescrape.LoadCommandConfig(
				arguments,
				environment(map[string]string{"SCRAPE_REQUEST_NATS_URL": "nats://nats:4222"}),
			); err == nil {
				t.Fatal("load command config: want an error")
			}
		})
	}
}

func TestLoadCommandConfigRefusesToPublishWithoutABroker(t *testing.T) {
	_, err := webarchivescrape.LoadCommandConfig([]string{
		"-pywb-url", "http://pywb:8080",
		"-pywb-collection", "archive",
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
	if _, err := webarchivescrape.LoadCommandConfig(
		[]string{"-coll", "archive"},
		environment(nil),
	); err == nil {
		t.Fatal("load command config: want an error")
	}
}

func loadedConfig(t *testing.T, arguments ...string) webarchivescrape.CommandConfig {
	t.Helper()
	cfg, err := webarchivescrape.LoadCommandConfig(arguments, environment(nil))
	if err != nil {
		t.Fatalf("load command config: %v", err)
	}
	return cfg
}

func environment(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}
