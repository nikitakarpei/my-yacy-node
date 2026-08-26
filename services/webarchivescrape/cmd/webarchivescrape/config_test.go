package main_test

import (
	"errors"
	"flag"
	"net/http"
	"strings"
	"testing"

	webarchivescrape "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/cmd/webarchivescrape"
	webarchivespywb "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/internal/webarchives/pywb"
)

func TestLoadCommandConfigReadsTheStatedQuery(t *testing.T) {
	cfg, err := webarchivescrape.LoadCommandConfig([]string{
		"-pywb-url", "http://pywb:8080",
		"-pywb-collection", "archive",
		"-url", "example.com",
		"-match-type", "host",
		"-from", "2024",
		"-to", "2025",
		"-page-limit", "50",
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
	query := theOnlyQuery(t, cfg)
	if query.URL != "example.com" || query.MatchType != "host" {
		t.Errorf("query = %+v", query)
	}
	if query.From != "2024" || query.To != "2025" {
		t.Errorf("query bounds = %+v", query)
	}
	if cfg.PageLimit != 50 {
		t.Errorf("page limit = %d, want 50", cfg.PageLimit)
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

	query := theOnlyQuery(t, cfg)
	if query.MediaType != "text/html" {
		t.Errorf("media type = %q, want text/html", query.MediaType)
	}
	if query.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want %d", query.StatusCode, http.StatusOK)
	}
}

func TestLoadCommandConfigSearchesAWholeDomainUnlessToldOtherwise(t *testing.T) {
	cfg := loadedConfig(t, "-pywb-url", "http://pywb:8080",
		"-pywb-collection", "archive", "-url", "example.com", "-dry-run")

	if matchType := theOnlyQuery(t, cfg).MatchType; matchType != "domain" {
		t.Errorf("match type = %q, want domain", matchType)
	}
}

func TestLoadCommandConfigNeedsNoBrokerForADryRun(t *testing.T) {
	cfg := loadedConfig(t, "-pywb-url", "http://pywb:8080",
		"-pywb-collection", "archive", "-url", "example.com", "-dry-run")

	if !cfg.DryRun {
		t.Error("dry run = false, want true")
	}
}

func TestLoadCommandConfigUsesNoPageLimitUnlessOneIsStated(t *testing.T) {
	cfg := loadedConfig(t, "-pywb-url", "http://pywb:8080",
		"-pywb-collection", "archive", "-url", "example.com", "-dry-run")

	if cfg.PageLimit != 0 {
		t.Errorf("page limit = %d, want all pages", cfg.PageLimit)
	}
}

func TestLoadCommandConfigRefusesANegativePageLimit(t *testing.T) {
	_, err := webarchivescrape.LoadCommandConfig([]string{
		"-pywb-url", "http://pywb:8080",
		"-pywb-collection", "archive",
		"-url", "example.com",
		"-page-limit", "-1",
		"-dry-run",
	}, environment(nil))
	if err == nil {
		t.Fatal("negative page limit should fail")
	}
	if !strings.Contains(err.Error(), "page-limit") {
		t.Fatalf("error = %v, want it to name page-limit", err)
	}
}

func TestLoadCommandConfigRefusesTheFormerCaptureLimit(t *testing.T) {
	_, err := webarchivescrape.LoadCommandConfig([]string{
		"-pywb-url", "http://pywb:8080",
		"-pywb-collection", "archive",
		"-url", "example.com",
		"-limit", "1",
		"-dry-run",
	}, environment(nil))
	if err == nil {
		t.Fatal("former capture limit should fail")
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

func TestLoadCommandConfigAsksAboutEveryStatedURL(t *testing.T) {
	cfg := loadedConfig(t, "-pywb-url", "http://pywb:8080", "-pywb-collection", "archive",
		"-url", "example.com", "-url", "example.org", "-match-type", "host", "-dry-run")

	if len(cfg.PywbCaptureQueries) != 2 {
		t.Fatalf("queries = %+v, want one per url", cfg.PywbCaptureQueries)
	}
	if cfg.PywbCaptureQueries[0].URL != "example.com" {
		t.Errorf("first query = %+v, want example.com", cfg.PywbCaptureQueries[0])
	}
	if cfg.PywbCaptureQueries[1].URL != "example.org" {
		t.Errorf("second query = %+v, want example.org", cfg.PywbCaptureQueries[1])
	}
	for _, query := range cfg.PywbCaptureQueries {
		if query.MatchType != "host" {
			t.Errorf("query = %+v, want every url matched the same way", query)
		}
	}
}

func TestLoadCommandConfigRefusesAnEmptyURL(t *testing.T) {
	_, err := webarchivescrape.LoadCommandConfig([]string{
		"-pywb-url", "http://pywb:8080",
		"-pywb-collection", "archive",
		"-url", "  ",
		"-dry-run",
	}, environment(nil))
	if err == nil {
		t.Fatal("empty url should fail")
	}
}

func TestLoadCommandConfigReportsARequestForHelpAsSuch(t *testing.T) {
	_, err := webarchivescrape.LoadCommandConfig([]string{"-h"}, environment(nil))

	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("err = %v, want a request for help", err)
	}
}

func theOnlyQuery(
	t *testing.T,
	cfg webarchivescrape.CommandConfig,
) webarchivespywb.CaptureQuery {
	t.Helper()
	if len(cfg.PywbCaptureQueries) != 1 {
		t.Fatalf("queries = %+v, want exactly one", cfg.PywbCaptureQueries)
	}
	return cfg.PywbCaptureQueries[0]
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
