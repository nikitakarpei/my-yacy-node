package main_test

import (
	"bytes"
	"context"
	"io"
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

func TestRunCommandPublishesTheNewestCaptureOfEveryPage(t *testing.T) {
	archiveURL := archiveServing(t, strings.Join(
		[]string{homePageRow, newerHomePageRow, aboutPageRow}, "\n",
	))

	published := &bytes.Buffer{}
	if err := webarchivescrape.RunCommand(
		context.Background(),
		dryRunConfig(t, archiveURL),
		published,
		io.Discard,
	); err != nil {
		t.Fatalf("run command: %v", err)
	}

	lines := writtenLines(published)
	if len(lines) != 2 {
		t.Fatalf("published = %q, want one line per page", published.String())
	}
	if !strings.Contains(lines[0], "20240501120000mp_") {
		t.Errorf("page = %q, want the newest capture of the home page", lines[0])
	}
	if !strings.Contains(lines[1], "https://example.com/about") {
		t.Errorf("page = %q, want the about page", lines[1])
	}
}

func TestRunCommandPublishesOneRunOfPagesForEveryStatedURL(t *testing.T) {
	otherHostRow := `{"urlkey": "org,example)/", "timestamp": "20240104150000", ` +
		`"url": "https://example.org/", "mime": "text/html", "status": "200"}`
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			rows := map[string]string{
				"example.com": homePageRow,
				"example.org": otherHostRow,
			}
			_, _ = writer.Write([]byte(rows[request.URL.Query().Get("url")]))
		},
	))
	t.Cleanup(server.Close)
	cfg := dryRunConfigFor(t, server.URL, "-url", "example.com", "-url", "example.org")

	published := &bytes.Buffer{}
	if err := webarchivescrape.RunCommand(
		context.Background(),
		cfg,
		published,
		io.Discard,
	); err != nil {
		t.Fatalf("run command: %v", err)
	}

	lines := writtenLines(published)
	if len(lines) != 2 {
		t.Fatalf("published = %q, want one line per stated url", published.String())
	}
	if !strings.Contains(lines[0], "https://example.com/") {
		t.Errorf("page = %q, want the page of the first url", lines[0])
	}
	if !strings.Contains(lines[1], "https://example.org/") {
		t.Errorf("page = %q, want the page of the second url", lines[1])
	}
}

func TestRunCommandReportsTheCapturesReadAndThePagesSelected(t *testing.T) {
	archiveURL := archiveServing(t, strings.Join(
		[]string{homePageRow, newerHomePageRow, aboutPageRow}, "\n",
	))

	report := &bytes.Buffer{}
	if err := webarchivescrape.RunCommand(
		context.Background(),
		dryRunConfig(t, archiveURL),
		io.Discard,
		report,
	); err != nil {
		t.Fatalf("run command: %v", err)
	}

	if !strings.Contains(report.String(), "read 3 captures, selected 2 pages") {
		t.Errorf("report = %q, want the captures read and the pages selected", report.String())
	}
}

func TestRunCommandReachesNoBrokerOnADryRun(t *testing.T) {
	cfg := dryRunConfig(t, archiveServing(t, homePageRow))
	cfg.ScrapeRequestNATSURL = "nats://127.0.0.1:1"

	published := &bytes.Buffer{}
	if err := webarchivescrape.RunCommand(
		context.Background(),
		cfg,
		published,
		io.Discard,
	); err != nil {
		t.Fatalf("run command: %v", err)
	}

	if len(writtenLines(published)) != 1 {
		t.Fatalf("published = %q, want the page it would publish", published.String())
	}
}

func TestRunCommandReportsThatTheArchiveHoldsMorePages(t *testing.T) {
	archiveURL := archiveServing(t, strings.Join(
		[]string{homePageRow, aboutPageRow}, "\n",
	))
	cfg := dryRunConfig(t, archiveURL)
	cfg.PageLimit = 1

	report := &bytes.Buffer{}
	if err := webarchivescrape.RunCommand(
		context.Background(),
		cfg,
		io.Discard,
		report,
	); err != nil {
		t.Fatalf("run command: %v", err)
	}

	if !strings.Contains(report.String(), "more pages") {
		t.Errorf("report = %q, want it to say pages were left behind", report.String())
	}
}

func TestRunCommandPublishesNothingWhenTheArchiveHoldsNoSuchPage(t *testing.T) {
	published := &bytes.Buffer{}
	if err := webarchivescrape.RunCommand(
		context.Background(),
		dryRunConfig(t, archiveServing(t, "")),
		published,
		io.Discard,
	); err != nil {
		t.Fatalf("run command: %v", err)
	}

	if published.String() != "" {
		t.Fatalf("published = %q, want none", published.String())
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

	published := &bytes.Buffer{}
	if err := webarchivescrape.RunCommand(
		context.Background(),
		cfg,
		published,
		io.Discard,
	); err != nil {
		t.Fatalf("run command: %v", err)
	}

	lines := writtenLines(published)
	if len(lines) != 2 {
		t.Fatalf("published = %q, want two distinct pages", published.String())
	}
	if !strings.Contains(lines[0], "20240501120000mp_") {
		t.Errorf("page = %q, want the newest capture of the home page", lines[0])
	}
	if !strings.Contains(lines[1], "https://example.com/about") {
		t.Errorf("page = %q, want the about page", lines[1])
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
		io.Discard,
	); err == nil {
		t.Fatal("run command: want an error")
	}
}

func TestRunCommandPublishesNothingWhenACaptureHasNoReplayURL(t *testing.T) {
	malformedPageRow := `{"urlkey":"com,example)/","timestamp":"20240101120000",` +
		`"url":"https://example.com/%zz"}`
	published := &bytes.Buffer{}

	err := webarchivescrape.RunCommand(
		context.Background(),
		dryRunConfig(t, archiveServing(t, malformedPageRow)),
		published,
		io.Discard,
	)

	if err == nil {
		t.Fatal("run command: want an error")
	}
	if published.String() != "" {
		t.Fatalf("published = %q, want none", published.String())
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
		io.Discard,
	); err == nil {
		t.Fatal("run command: want an error")
	}
}

func writtenLines(written *bytes.Buffer) []string {
	return strings.Split(strings.TrimSuffix(written.String(), "\n"), "\n")
}

func dryRunConfig(t *testing.T, archiveURL string) webarchivescrape.CommandConfig {
	t.Helper()
	return dryRunConfigFor(t, archiveURL, "-url", "example.com")
}

func dryRunConfigFor(
	t *testing.T,
	archiveURL string,
	queries ...string,
) webarchivescrape.CommandConfig {
	t.Helper()
	arguments := append(
		[]string{"-pywb-url", archiveURL, "-pywb-collection", "archive", "-dry-run"},
		queries...,
	)
	cfg, err := webarchivescrape.LoadCommandConfig(arguments, func(string) string { return "" })
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
