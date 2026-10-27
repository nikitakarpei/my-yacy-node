package searchpages_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pageshell"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchpages"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchqueries"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchresults"
)

type fakeSource struct {
	asked    *[]searchqueries.Search
	page     searchresults.SourcePage
	answered bool
}

func (s fakeSource) PageFor(
	_ context.Context,
	search searchqueries.Search,
) (searchresults.SourcePage, bool) {
	if s.asked != nil {
		*s.asked = append(*s.asked, search)
	}
	return s.page, s.answered
}

type prefixedResultLinks struct{}

func (prefixedResultLinks) LinkFor(page string) string {
	return "/visit?url=" + page
}

type recordingObserver struct {
	failures []error
	lock     sync.Mutex
}

func (o *recordingObserver) PageRenderFailed(_ context.Context, err error) {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.failures = append(o.failures, err)
}

func TestTheFrontPageShowsOnlyTheSearchBoxAndItsOperators(t *testing.T) {
	var asked []searchqueries.Search
	handler := searchPagesWith(
		fakeSource{asked: &asked, answered: true},
		fakeSource{asked: &asked, answered: true},
	)

	page := bodyOf(t, handler, "/")

	assertContains(t, page, `name="q"`, "<title>Search</title>", "!ct", "!ya", ":de")
	assertLacks(t, page, "No results.", `class="result"`)
	if len(asked) != 0 {
		t.Errorf("sources were asked %v, want no search", asked)
	}
}

func TestTheResultsPageMergesTheSourcesThroughTheResultLinks(t *testing.T) {
	handler := searchPagesWith(
		fakeSource{answered: true, page: searchresults.SourcePage{{
			Title: "Crawled", Address: "https://www.crawled.example/", Snippet: "about yacy",
		}}},
		fakeSource{answered: true, page: searchresults.SourcePage{{
			Title: "Network", Address: "https://network.example/page", Snippet: "YaCy & peers",
		}}},
	)

	page := bodyOf(t, handler, "/search?q=yacy")

	assertContains(t, page,
		"<title>yacy</title>",
		`href="/visit?url=https://www.crawled.example/"`,
		">crawled.example<",
		">network.example/page<",
		"about <b>yacy</b>",
		"<b>YaCy</b> &amp; peers",
	)
	if strings.Index(page, "Crawled") > strings.Index(page, "Network") {
		t.Error("network result shows before the crawled result, want the crawled one first")
	}
	assertLacks(t, page, "No results.", `rel="next"`, `rel="prev"`)
}

func TestTheSearchAsksOnlyTheSourcesOfItsOperators(t *testing.T) {
	var crawledAsks, networkAsks []searchqueries.Search
	handler := searchPagesWith(
		fakeSource{asked: &crawledAsks, answered: true},
		fakeSource{asked: &networkAsks, answered: true},
	)

	bodyOf(t, handler, "/search?q=!ct+yacy")

	if len(crawledAsks) != 1 || len(networkAsks) != 0 {
		t.Errorf("asked crawled %d times and network %d times, want crawled once",
			len(crawledAsks), len(networkAsks))
	}
}

func TestASearchWithoutResultsSaysSo(t *testing.T) {
	handler := searchPagesWith(fakeSource{answered: true}, fakeSource{answered: true})

	assertContains(t, bodyOf(t, handler, "/search?q=nothing"), "No results.")
}

func TestASourceThatFailedHasANoticeAboveTheOtherResults(t *testing.T) {
	handler := searchPagesWith(
		fakeSource{answered: true, page: searchresults.SourcePage{{
			Title: "Crawled", Address: "https://crawled.example/",
		}}},
		fakeSource{answered: false},
	)

	page := bodyOf(t, handler, "/search?q=yacy")

	assertContains(t, page, "Failed to search the YaCy network.", "Crawled")
	assertLacks(t, page, "Failed to search your crawled pages.")
}

func TestASearchWhoseSourcesAllFailedShowsOnlyTheNotices(t *testing.T) {
	handler := searchPagesWith(fakeSource{answered: false}, fakeSource{answered: false})

	page := bodyOf(t, handler, "/search?q=yacy")

	assertContains(t, page,
		"Failed to search your crawled pages.", "Failed to search the YaCy network.")
	assertLacks(t, page, "No results.")
}

func TestAFullSourcePageLinksTheNextPageAndALaterPageThePreviousOne(t *testing.T) {
	fullPage := make(searchresults.SourcePage, 0, searchresults.ResultsPerSourcePage)
	for position := range searchresults.ResultsPerSourcePage {
		fullPage = append(fullPage, searchresults.Result{
			Address: fmt.Sprintf("https://example.org/%d", position),
		})
	}
	handler := searchPagesWith(fakeSource{answered: true, page: fullPage}, fakeSource{})

	firstPage := bodyOf(t, handler, "/search?q=!ct+yacy")
	secondPage := bodyOf(t, handler, "/search?q=!ct+yacy&page=2")
	thirdPage := bodyOf(t, handler, "/search?q=!ct+yacy&page=3")

	assertContains(t, firstPage, `href="/search?page=2&amp;q=%21ct&#43;yacy" rel="next"`)
	assertLacks(t, firstPage, `rel="prev"`)
	assertContains(t, secondPage,
		"<title>!ct yacy – page 2</title>",
		`href="/search?q=%21ct&#43;yacy" rel="prev"`,
	)
	assertContains(t, thirdPage, `href="/search?page=2&amp;q=%21ct&#43;yacy" rel="prev"`)
}

func TestAShortWordIsNotHighlighted(t *testing.T) {
	handler := searchPagesWith(
		fakeSource{answered: true, page: searchresults.SourcePage{{
			Address: "https://example.org/", Snippet: "a yacy peer",
		}}},
		fakeSource{answered: true},
	)

	assertContains(t, bodyOf(t, handler, "/search?q=a+yacy"), "a <b>yacy</b> peer")
}

func TestASourceThatIsNotOfferedHasNoOperatorAndItsShortcutIsAQueryWord(t *testing.T) {
	var asked []searchqueries.Search
	handler := searchpages.New(
		map[searchqueries.Source]searchpages.ResultSource{
			searchqueries.YaCyNetwork: fakeSource{asked: &asked, answered: true},
		},
		prefixedResultLinks{},
		pageshell.TabBar{},
		&recordingObserver{},
	)

	page := bodyOf(t, handler, "/search?q=!ct+yacy")

	assertContains(t, page, "!ya")
	assertLacks(t, page, "<dt>!ct</dt>", "crawled pages")
	if len(asked) != 1 || asked[0].Text() != "!ct yacy" {
		t.Errorf("asked %v, want the YaCy network asked for the shortcut as a word", asked)
	}
}

func TestTheOpenSearchDescriptionSearchesThisPage(t *testing.T) {
	handler := searchPagesWith(fakeSource{}, fakeSource{})

	description := bodyOf(t, handler, "/opensearch.xml")

	assertContains(t, description,
		"<ShortName>search.example:8097</ShortName>",
		`template="http://search.example:8097/search?q={searchTerms}"`,
	)
}

func TestTheOpenSearchDescriptionBehindAnHTTPSProxySearchesOverHTTPS(t *testing.T) {
	handler := searchPagesWith(fakeSource{}, fakeSource{})
	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"http://search.example/opensearch.xml",
		nil,
	)
	request.Header.Set("X-Forwarded-Proto", "https")
	answer := httptest.NewRecorder()

	handler.ServeHTTP(answer, request)

	assertContains(t, answer.Body.String(), `template="https://search.example/search?q=`)
}

func TestThePageServesItsStyleSheet(t *testing.T) {
	handler := searchPagesWith(fakeSource{}, fakeSource{})

	assertContains(t, bodyOf(t, handler, "/static/search.css"), ".result")
}

func TestThePageLinksTheShellStylesheetAndServesIt(t *testing.T) {
	handler := searchPagesWith(fakeSource{}, fakeSource{})

	assertContains(t, bodyOf(t, handler, "/"), `href="`+pageshell.StylesheetPath+`"`)
	assertContains(t, bodyOf(t, handler, pageshell.StylesheetPath), "--shell-accent")
}

func TestEveryPageShowsTheTabBar(t *testing.T) {
	handler := searchpages.New(
		map[searchqueries.Source]searchpages.ResultSource{
			searchqueries.YaCyNetwork: fakeSource{answered: true},
		},
		prefixedResultLinks{},
		pageshell.TabBar{Tabs: []pageshell.Tab{
			{Title: "Search", URL: "http://search.example", Current: true},
			{Title: "Crawls", URL: "http://crawls.example"},
		}},
		&recordingObserver{},
	)

	for _, target := range []string{"/", "/search?q=yacy"} {
		assertContains(t, bodyOf(t, handler, target),
			`<a href="http://search.example" aria-current="page">Search</a>`,
			`<a href="http://crawls.example">Crawls</a>`,
		)
	}
}

func TestAPageWithoutATabBarShowsNone(t *testing.T) {
	handler := searchPagesWith(fakeSource{}, fakeSource{})

	assertLacks(t, bodyOf(t, handler, "/"), "shell-tabs")
}

func searchPagesWith(crawledPages, yacyNetwork fakeSource) http.Handler {
	return searchpages.New(
		map[searchqueries.Source]searchpages.ResultSource{
			searchqueries.CrawledPages: crawledPages,
			searchqueries.YaCyNetwork:  yacyNetwork,
		},
		prefixedResultLinks{},
		pageshell.TabBar{},
		&recordingObserver{},
	)
}

func bodyOf(t *testing.T, handler http.Handler, target string) string {
	t.Helper()
	answer := httptest.NewRecorder()
	handler.ServeHTTP(
		answer,
		httptest.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			"http://search.example:8097"+target,
			nil,
		),
	)
	if answer.Code != http.StatusOK {
		t.Fatalf("GET %s answered %d", target, answer.Code)
	}
	body, err := io.ReadAll(answer.Body)
	if err != nil {
		t.Fatalf("read %s: %v", target, err)
	}
	return string(body)
}

func assertContains(t *testing.T, body string, wantedTexts ...string) {
	t.Helper()
	for _, text := range wantedTexts {
		if !strings.Contains(body, text) {
			t.Errorf("body lacks %q:\n%s", text, body)
		}
	}
}

func assertLacks(t *testing.T, body string, unwantedTexts ...string) {
	t.Helper()
	for _, text := range unwantedTexts {
		if strings.Contains(body, text) {
			t.Errorf("body holds %q:\n%s", text, body)
		}
	}
}
