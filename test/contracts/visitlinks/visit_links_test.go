package visitlinks_test

import (
	"context"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	searchpage "github.com/nikitakarpei/yacy-rwi-node/searchpage/assembly"
	visitcrawl "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/assembly"
)

const (
	linkSecret  = "shared-secret"
	resultPage  = "https://crawled.example/page?a=1&b=2"
	callTimeout = 5 * time.Second
)

var resultLink = regexp.MustCompile(`<h2><a href="([^"]+)"`)

func TestAResultLinkOfTheSearchPageOpensThroughVisitCrawl(t *testing.T) {
	visitCrawlAddress := runVisitCrawl(t)
	searchPageAddress := runSearchPage(t, visitCrawlAddress)

	link := resultLinkOn(t, "http://"+searchPageAddress+"/search?q=!ct+page")
	destination := redirectOf(t, "http://"+searchPageAddress+link)

	if destination != resultPage {
		t.Errorf("result link redirected to %q, want %q", destination, resultPage)
	}
}

func runVisitCrawl(t *testing.T) string {
	t.Helper()
	settings := map[string]string{
		visitcrawl.EnvCrawlNATSURL: natstestserver.Start(t),
		visitcrawl.EnvLinkSecret:   linkSecret,
		visitcrawl.EnvListenAddr:   "127.0.0.1:0",
	}
	cfg, err := visitcrawl.LoadServiceConfig(func(key string) string { return settings[key] })
	if err != nil {
		t.Fatalf("load the visitcrawl config: %v", err)
	}
	listeners, err := visitcrawl.ListenersFor(t.Context(), cfg)
	if err != nil {
		t.Fatalf("listen for visitcrawl: %v", err)
	}
	runUntilCleanup(t, func(ctx context.Context) error {
		return visitcrawl.RunService(ctx, cfg, listeners, prometheus.NewRegistry())
	})
	return listeners.Intake.Addr().String()
}

func runSearchPage(t *testing.T, visitCrawlAddress string) string {
	t.Helper()
	corpusText := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"matches":[{"url":"`+resultPage+`","title":"Page"}]}`)
	}))
	t.Cleanup(corpusText.Close)
	settings := map[string]string{
		searchpage.EnvCorpusTextURL:   corpusText.URL,
		searchpage.EnvYaCySearchURL:   "http://127.0.0.1:1",
		searchpage.EnvVisitCrawlURL:   "http://" + visitCrawlAddress,
		searchpage.EnvVisitLinkSecret: linkSecret,
		searchpage.EnvListenAddr:      "127.0.0.1:0",
	}
	cfg, err := searchpage.LoadServiceConfig(func(key string) string { return settings[key] })
	if err != nil {
		t.Fatalf("load the searchpage config: %v", err)
	}
	listeners, err := searchpage.ListenersFor(t.Context(), cfg)
	if err != nil {
		t.Fatalf("listen for searchpage: %v", err)
	}
	runUntilCleanup(t, func(ctx context.Context) error {
		return searchpage.RunService(ctx, cfg, listeners, prometheus.NewRegistry())
	})
	return listeners.Page.Addr().String()
}

func runUntilCleanup(t *testing.T, run func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("service stopped with %v", err)
		}
	})
}

func resultLinkOn(t *testing.T, pageURL string) string {
	t.Helper()
	answer := answerTo(t, http.DefaultClient, pageURL)
	defer func() { _ = answer.Body.Close() }()
	page, err := io.ReadAll(answer.Body)
	if err != nil {
		t.Fatalf("read the results page: %v", err)
	}
	match := resultLink.FindSubmatch(page)
	if match == nil {
		t.Fatalf("results page holds no result link:\n%s", page)
	}
	return html.UnescapeString(string(match[1]))
}

func redirectOf(t *testing.T, visitURL string) string {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	answer := answerTo(t, client, visitURL)
	defer func() { _ = answer.Body.Close() }()
	return answer.Header.Get("Location")
}

func answerTo(t *testing.T, client *http.Client, target string) *http.Response {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), callTimeout)
	t.Cleanup(cancel)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	answer, err := client.Do(request)
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	return answer
}
