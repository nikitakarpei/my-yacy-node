package consolepages_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/consolemessages"
	"github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/consolepages"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

var (
	startedAt = time.Date(2026, 10, 21, 9, 30, 0, 0, time.UTC)
	now       = startedAt.Add(2*time.Hour + 5*time.Minute)
)

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return now
}

type console struct {
	handler  http.Handler
	observer *recordingObserver
}

func consoleOver(crawler *fakeCrawler) console {
	return consoleWithTabBar(crawler, webuistyle.TabBar{})
}

func consoleWithTabBar(crawler *fakeCrawler, tabBar webuistyle.TabBar) console {
	observer := &recordingObserver{}
	return console{
		handler: consolepages.New(
			crawler,
			consolemessages.English(),
			tabBar,
			fixedClock{},
			observer,
		),
		observer: observer,
	}
}

func (console console) get(path string) *httptest.ResponseRecorder {
	answer := httptest.NewRecorder()
	console.handler.ServeHTTP(answer,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
	return answer
}

func (console console) post(path string, fields url.Values) *httptest.ResponseRecorder {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path,
		strings.NewReader(fields.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	answer := httptest.NewRecorder()
	console.handler.ServeHTTP(answer, request)
	return answer
}

func seedURLOf(t *testing.T, rawURL string) canonicalurl.CanonicalURL {
	t.Helper()
	seedURL, err := canonicalurl.CanonicalURLOf(rawURL)
	if err != nil {
		t.Fatalf("CanonicalURLOf(%q): %v", rawURL, err)
	}
	return seedURL
}

func runningCrawlOf(t *testing.T, crawlID string) yacycrawlcontract.Crawl {
	t.Helper()
	seedURL := seedURLOf(t, "https://example.org/docs/")
	return yacycrawlcontract.Crawl{
		CrawlID:           crawlID,
		Label:             "docs site",
		StartedAt:         startedAt,
		State:             yacycrawlcontract.CrawlStateRunning,
		FirstSeedURL:      seedURL,
		AmountOfSeedURLs:  3,
		PendingPageVisits: 5,
		UnadmittedLinks:   11,
		Order: &yacycrawlcontract.CrawlOrder{
			OrderID: crawlID,
			Label:   "docs site",
			SeedURLs: []canonicalurl.CanonicalURL{
				seedURL,
				seedURLOf(t, "https://example.org/blog/"),
			},
			Profile: yacycrawlcontract.CrawlProfile{
				Scope:           yacycrawlcontract.ScopeDomain,
				MaxDepth:        3,
				MaxPagesPerHost: 50,
			},
		},
	}
}

func finishedCrawlOf(t *testing.T, crawlID string) yacycrawlcontract.Crawl {
	t.Helper()
	crawl := runningCrawlOf(t, crawlID)
	crawl.Label = ""
	crawl.State = yacycrawlcontract.CrawlStateSucceeded
	crawl.FinishedAt = startedAt.Add(time.Hour)
	crawl.PendingPageVisits = 0
	return crawl
}

func assertStatus(t *testing.T, answer *httptest.ResponseRecorder, status int) {
	t.Helper()
	if answer.Code != status {
		t.Fatalf("status = %d, want %d; body:\n%s", answer.Code, status, answer.Body)
	}
}

func assertShows(t *testing.T, answer *httptest.ResponseRecorder, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(answer.Body.String(), fragment) {
			t.Errorf("page lacks %q; body:\n%s", fragment, answer.Body)
		}
	}
}

func assertHides(t *testing.T, answer *httptest.ResponseRecorder, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if strings.Contains(answer.Body.String(), fragment) {
			t.Errorf("page shows %q; body:\n%s", fragment, answer.Body)
		}
	}
}

func assertRedirect(t *testing.T, answer *httptest.ResponseRecorder, location string) {
	t.Helper()
	assertStatus(t, answer, http.StatusSeeOther)
	if got := answer.Header().Get("Location"); got != location {
		t.Fatalf("Location = %q, want %q", got, location)
	}
}

func TestConsoleRefusesAFormFromAnotherOrigin(t *testing.T) {
	crawler := &fakeCrawler{}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/crawler-pause",
		strings.NewReader(url.Values{"Paused": {"true"}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	answer := httptest.NewRecorder()

	consoleOver(crawler).handler.ServeHTTP(answer, request)

	assertStatus(t, answer, http.StatusForbidden)
	if len(crawler.calls) != 0 {
		t.Fatalf("crawler calls = %v, want none", crawler.calls)
	}
}

func TestConsoleServesItsStylesheet(t *testing.T) {
	answer := consoleOver(&fakeCrawler{}).get("/static/console.css")

	assertStatus(t, answer, http.StatusOK)
	if contentType := answer.Header().
		Get("Content-Type"); !strings.HasPrefix(
		contentType,
		"text/css",
	) {
		t.Fatalf("Content-Type = %q, want text/css", contentType)
	}
}

func TestConsoleLinksTheWebUIStylesheetAndServesIt(t *testing.T) {
	console := consoleOver(&fakeCrawler{})

	assertShows(t, console.get("/"), `href="`+webuistyle.StylesheetPath+`"`)
	assertShows(t, console.get(webuistyle.StylesheetPath), "--webui-accent")
}

func TestEveryConsolePageShowsTheTabBar(t *testing.T) {
	crawler := &fakeCrawler{
		crawls: map[string]yacycrawlcontract.Crawl{"c1": runningCrawlOf(t, "c1")},
	}
	console := consoleWithTabBar(crawler, webuistyle.TabBar{Tabs: []webuistyle.Tab{
		{Title: "Search", URL: "http://search.example"},
		{Title: "Crawls", URL: "http://crawls.example", Current: true},
	}})

	for _, path := range []string{"/", "/crawls/c1", "/crawls/new"} {
		assertShows(t, console.get(path),
			`<a href="http://search.example">Search</a>`,
			`<a href="http://crawls.example" aria-current="page">Crawls</a>`,
		)
	}
}

func TestAConsoleWithoutATabBarShowsNone(t *testing.T) {
	assertHides(t, consoleOver(&fakeCrawler{}).get("/"), "webui-tabs")
}
