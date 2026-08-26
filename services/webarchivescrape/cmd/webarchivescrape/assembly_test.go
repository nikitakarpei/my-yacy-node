package main_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webarchivescrape "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/cmd/webarchivescrape"
)

const (
	homePageRow = `{"urlkey": "com,example)/", "timestamp": "20240101120000", ` +
		`"url": "https://example.com/", "mime": "text/html", "status": "200"}`
	newerHomePageRow = `{"urlkey": "com,example)/", "timestamp": "20240501120000", ` +
		`"url": "https://example.com/", "mime": "text/html", "status": "200"}`
	aboutPageRow = `{"urlkey": "com,example)/about", "timestamp": "20240101120000", ` +
		`"url": "https://example.com/about", "mime": "text/html", "status": "200"}`
)

func TestRunCommandWritesOneScrapeRequestPerNewestCapture(t *testing.T) {
	archiveURL := archiveServing(t, strings.Join(
		[]string{homePageRow, newerHomePageRow, aboutPageRow}, "\n",
	))

	requests := &bytes.Buffer{}
	if err := webarchivescrape.RunCommand(
		context.Background(),
		dryRunConfig(t, archiveURL),
		requests,
	); err != nil {
		t.Fatalf("run command: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(requests.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("requests = %q, want one per page", requests.String())
	}
	if !strings.Contains(lines[0], "20240501120000mp_") {
		t.Errorf("request = %q, want the newest capture of the home page", lines[0])
	}
	if !strings.Contains(lines[1], "https://example.com/about") {
		t.Errorf("request = %q, want the about page", lines[1])
	}
}

func TestRunCommandWritesNothingWhenTheArchiveHoldsNoSuchPage(t *testing.T) {
	requests := &bytes.Buffer{}
	if err := webarchivescrape.RunCommand(
		context.Background(),
		dryRunConfig(t, archiveServing(t, "")),
		requests,
	); err != nil {
		t.Fatalf("run command: %v", err)
	}

	if requests.String() != "" {
		t.Fatalf("requests = %q, want none", requests.String())
	}
}

func TestRunCommandLimitsDistinctPagesAfterChoosingNewestCaptures(t *testing.T) {
	contactPageRow := `{"urlkey": "com,example)/contact", ` +
		`"timestamp": "20240101120000", ` +
		`"url": "https://example.com/contact", "mime": "text/html", "status": "200"}`
	archiveURL := archiveServing(t, strings.Join(
		[]string{homePageRow, newerHomePageRow, aboutPageRow, contactPageRow}, "\n",
	))
	cfg := dryRunConfig(t, archiveURL)
	cfg.PageLimit = 2

	requests := &bytes.Buffer{}
	if err := webarchivescrape.RunCommand(context.Background(), cfg, requests); err != nil {
		t.Fatalf("run command: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(requests.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("requests = %q, want two distinct pages", requests.String())
	}
	if !strings.Contains(lines[0], "20240501120000mp_") {
		t.Errorf("request = %q, want the newest capture of the home page", lines[0])
	}
	if !strings.Contains(lines[1], "https://example.com/about") {
		t.Errorf("request = %q, want the about page", lines[1])
	}
}

func TestRunCommandFailsWhenTheArchiveRefusesTheQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusInternalServerError)
		},
	))
	t.Cleanup(server.Close)

	if err := webarchivescrape.RunCommand(
		context.Background(),
		dryRunConfig(t, server.URL),
		&bytes.Buffer{},
	); err == nil {
		t.Fatal("run command: want an error")
	}
}

func TestRunCommandWritesNothingWhenACaptureHasNoReplayURL(t *testing.T) {
	malformedPageRow := `{"urlkey":"com,example)/","timestamp":"20240101120000",` +
		`"url":"https://example.com/%zz"}`
	requests := &bytes.Buffer{}

	err := webarchivescrape.RunCommand(
		context.Background(),
		dryRunConfig(t, archiveServing(t, malformedPageRow)),
		requests,
	)

	if err == nil {
		t.Fatal("run command: want an error")
	}
	if requests.String() != "" {
		t.Fatalf("requests = %q, want none", requests.String())
	}
}

func TestRunCommandFailsWhenTheBrokerIsUnreachable(t *testing.T) {
	cfg := dryRunConfig(t, archiveServing(t, homePageRow))
	cfg.DryRun = false
	cfg.ScrapeRequestNATSURL = "nats://127.0.0.1:1"

	if err := webarchivescrape.RunCommand(
		context.Background(),
		cfg,
		&bytes.Buffer{},
	); err == nil {
		t.Fatal("run command: want an error")
	}
}

func dryRunConfig(t *testing.T, archiveURL string) webarchivescrape.CommandConfig {
	t.Helper()
	cfg, err := webarchivescrape.LoadCommandConfig([]string{
		"-pywb-url", archiveURL,
		"-pywb-collection", "archive",
		"-url", "example.com",
		"-dry-run",
	}, func(string) string { return "" })
	if err != nil {
		t.Fatalf("load command config: %v", err)
	}
	return cfg
}

func archiveServing(t *testing.T, rows string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte(rows))
		},
	))
	t.Cleanup(server.Close)
	return server.URL
}
