package searchpages_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/searchmessages"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/searchpages"
)

type searchCall struct {
	query  string
	offset int
	limit  int
}

type fakeSearchEngine struct {
	results []searchpages.SearchResult
	err     error
	calls   []searchCall
}

func (engine *fakeSearchEngine) SearchResultsFor(
	_ context.Context,
	query string,
	offset int,
	limit int,
) ([]searchpages.SearchResult, error) {
	engine.calls = append(engine.calls, searchCall{query: query, offset: offset, limit: limit})
	return engine.results, engine.err
}

type recordingObserver struct {
	engineCallFailures []error
}

func (observer *recordingObserver) SearchEngineCallFailed(_ context.Context, err error) {
	observer.engineCallFailures = append(observer.engineCallFailures, err)
}

func (observer *recordingObserver) PageRenderFailed(context.Context, error) {}

type addressLinks struct{}

func (addressLinks) LinkFor(address string) string {
	return address
}

type visitLinks struct{}

func (visitLinks) LinkFor(address string) string {
	return "/visit?url=" + address
}

func pageOver(engine *fakeSearchEngine, observer *recordingObserver) http.Handler {
	return searchpages.New(engine, addressLinks{}, searchmessages.English(), observer)
}

func get(handler http.Handler, path string) *httptest.ResponseRecorder {
	answer := httptest.NewRecorder()
	handler.ServeHTTP(answer,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
	return answer
}

func resultsNamed(amount int) []searchpages.SearchResult {
	results := make([]searchpages.SearchResult, 0, amount)
	for index := range amount {
		results = append(results, searchpages.SearchResult{
			Title:   fmt.Sprintf("Result %d", index),
			Address: fmt.Sprintf("https://example.org/%d", index),
		})
	}
	return results
}

func TestPageWithoutAQueryShowsTheSearchBoxAndAsksNothing(t *testing.T) {
	engine := &fakeSearchEngine{}

	answer := get(pageOver(engine, &recordingObserver{}), "/")

	if answer.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", answer.Code)
	}
	if !strings.Contains(answer.Body.String(), `name="query"`) {
		t.Errorf("page lacks the search box:\n%s", answer.Body)
	}
	if len(engine.calls) != 0 {
		t.Errorf("engine asked %v, want no call", engine.calls)
	}
}

func TestPageShowsTheResultsOfTheQuery(t *testing.T) {
	engine := &fakeSearchEngine{results: []searchpages.SearchResult{{
		Title:       "YaCy",
		Address:     "https://yacy.net/",
		Description: "Decentralized web search",
		PublishedAt: time.Date(2026, 10, 21, 9, 30, 0, 0, time.UTC),
	}}}

	page := get(pageOver(engine, &recordingObserver{}), "/?query=+yacy+").Body.String()

	for _, want := range []string{
		`href="https://yacy.net/"`, ">YaCy</a>", "Decentralized web search", "2026-10-21",
		`value="yacy"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q:\n%s", want, page)
		}
	}
	if want := (searchCall{query: "yacy", offset: 0, limit: 10}); engine.calls[0] != want {
		t.Errorf("engine call = %+v, want %+v", engine.calls[0], want)
	}
}

func TestPageSaysWhenNoResultMatches(t *testing.T) {
	page := get(
		pageOver(&fakeSearchEngine{}, &recordingObserver{}),
		"/?query=nothing",
	).Body.String()

	if !strings.Contains(page, "No results for “nothing”.") {
		t.Errorf("page lacks the empty notice:\n%s", page)
	}
}

func TestPageLinksTheNextPageWhenThePageIsFull(t *testing.T) {
	engine := &fakeSearchEngine{results: resultsNamed(10)}

	page := get(pageOver(engine, &recordingObserver{}), "/?query=yacy").Body.String()

	if !strings.Contains(page, `href="/?query=yacy&amp;startRecord=10"`) {
		t.Errorf("page lacks the link to the next page:\n%s", page)
	}
	if strings.Contains(page, "Previous") {
		t.Errorf("first page links a previous page:\n%s", page)
	}
}

func TestPageAsksForTheResultsFromTheStartRecord(t *testing.T) {
	engine := &fakeSearchEngine{results: resultsNamed(3)}

	page := get(pageOver(engine, &recordingObserver{}), "/?query=yacy&startRecord=20").Body.String()

	if engine.calls[0].offset != 20 {
		t.Errorf("engine offset = %d, want 20", engine.calls[0].offset)
	}
	if !strings.Contains(page, `href="/?query=yacy&amp;startRecord=10"`) {
		t.Errorf("page lacks the link to the previous page:\n%s", page)
	}
	if strings.Contains(page, "Next") {
		t.Errorf("last page links a next page:\n%s", page)
	}
}

func TestPageStartsFromTheFirstResultForAnUnusableStartRecord(t *testing.T) {
	for _, startRecord := range []string{"-5", "many"} {
		engine := &fakeSearchEngine{}

		get(pageOver(engine, &recordingObserver{}), "/?query=yacy&startRecord="+startRecord)

		if engine.calls[0].offset != 0 {
			t.Errorf("offset for startRecord %q = %d, want 0", startRecord, engine.calls[0].offset)
		}
	}
}

func TestPageTellsWhenTheSearchEngineDoesNotAnswer(t *testing.T) {
	observer := &recordingObserver{}
	engine := &fakeSearchEngine{err: errors.New("connection refused")}

	answer := get(pageOver(engine, observer), "/?query=yacy")

	if answer.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", answer.Code)
	}
	if !strings.Contains(answer.Body.String(), "The search service does not answer.") {
		t.Errorf("page lacks the notice:\n%s", answer.Body)
	}
	if len(observer.engineCallFailures) != 1 {
		t.Errorf("observed failures = %v, want one", observer.engineCallFailures)
	}
}

func TestPageDoesNotLinkAResultWithAScriptAddress(t *testing.T) {
	engine := &fakeSearchEngine{results: []searchpages.SearchResult{{
		Title:   "Trap",
		Address: "javascript:alert(1)",
	}}}

	page := get(pageOver(engine, &recordingObserver{}), "/?query=yacy").Body.String()

	if strings.Contains(page, `href="javascript:`) {
		t.Errorf("page links a script address:\n%s", page)
	}
}

func TestPageServesItsStylesheet(t *testing.T) {
	answer := get(pageOver(&fakeSearchEngine{}, &recordingObserver{}), "/static/search.css")

	if answer.Code != http.StatusOK || !strings.Contains(answer.Body.String(), ".result") {
		t.Errorf("stylesheet answered %d:\n%s", answer.Code, answer.Body)
	}
}

func TestPageLinksEachResultThroughItsResultLinks(t *testing.T) {
	engine := &fakeSearchEngine{results: []searchpages.SearchResult{{
		Title:   "YaCy",
		Address: "https://yacy.net/",
	}}}
	page := searchpages.New(engine, visitLinks{}, searchmessages.English(), &recordingObserver{})

	body := get(page, "/?query=yacy").Body.String()

	if !strings.Contains(body, `href="/visit?url=https://yacy.net/"`) {
		t.Errorf("page lacks the result link:\n%s", body)
	}
	if !strings.Contains(body, `<span class="url">https://yacy.net/</span>`) {
		t.Errorf("page lacks the result address:\n%s", body)
	}
}
