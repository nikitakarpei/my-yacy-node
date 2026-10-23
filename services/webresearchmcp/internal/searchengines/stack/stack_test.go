package stack_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/searchengines/stack"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/websearch"
)

const searchQuery = "research subject"

type answeringEngine struct {
	searchResults []websearch.SearchResult
	failure       error
}

func (e answeringEngine) SearchResultsFor(
	context.Context,
	string,
) ([]websearch.SearchResult, error) {
	return e.searchResults, e.failure
}

type engineFailure struct {
	engineName string
	query      string
}

type recordingObserver struct {
	mu       sync.Mutex
	failures []engineFailure
}

func (o *recordingObserver) EngineFailed(
	_ context.Context,
	engineName string,
	query string,
	_ error,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.failures = append(o.failures, engineFailure{engineName: engineName, query: query})
}

func resultsAt(urls ...string) []websearch.SearchResult {
	results := make([]websearch.SearchResult, 0, len(urls))
	for _, url := range urls {
		results = append(results, websearch.SearchResult{URL: url})
	}
	return results
}

func urlsOf(results []websearch.SearchResult) []string {
	urls := make([]string, 0, len(results))
	for _, result := range results {
		urls = append(urls, result.URL)
	}
	return urls
}

func TestSearchInterleavesTheResultsOfEveryEngineByRank(t *testing.T) {
	searchStack := stack.NewStack([]stack.Engine{
		{Name: "corpustext", Engine: answeringEngine{searchResults: resultsAt(
			"https://a.example/1", "https://a.example/2", "https://a.example/3",
		)}},
		{Name: "yacy", Engine: answeringEngine{searchResults: resultsAt(
			"https://b.example/1",
		)}},
	}, &recordingObserver{})

	results, err := searchStack.SearchResultsFor(context.Background(), searchQuery)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	want := []string{
		"https://a.example/1", "https://b.example/1", "https://a.example/2", "https://a.example/3",
	}
	if got := urlsOf(results); !slices.Equal(got, want) {
		t.Errorf("results = %v, want %v", got, want)
	}
}

func TestSearchAnswersWithEachPageOnce(t *testing.T) {
	searchStack := stack.NewStack([]stack.Engine{
		{Name: "corpustext", Engine: answeringEngine{searchResults: resultsAt(
			"https://a.example/page", "https://a.example/other",
		)}},
		{Name: "yacy", Engine: answeringEngine{searchResults: resultsAt(
			"HTTPS://A.EXAMPLE/page", "not a url", "not a url",
		)}},
	}, &recordingObserver{})

	results, err := searchStack.SearchResultsFor(context.Background(), searchQuery)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	want := []string{"https://a.example/page", "https://a.example/other", "not a url"}
	if got := urlsOf(results); !slices.Equal(got, want) {
		t.Errorf("results = %v, want %v", got, want)
	}
}

func TestSearchLeavesOutAnEngineThatFailsAndReportsIt(t *testing.T) {
	observer := &recordingObserver{}
	searchStack := stack.NewStack([]stack.Engine{
		{Name: "corpustext", Engine: answeringEngine{searchResults: resultsAt(
			"https://a.example/1",
		)}},
		{Name: "yacy", Engine: answeringEngine{failure: errors.New("away")}},
	}, observer)

	results, err := searchStack.SearchResultsFor(context.Background(), searchQuery)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if got := urlsOf(results); !slices.Equal(got, []string{"https://a.example/1"}) {
		t.Errorf("results = %v, want the corpustext result", got)
	}
	want := []engineFailure{{engineName: "yacy", query: searchQuery}}
	if !slices.Equal(observer.failures, want) {
		t.Errorf("failures = %v, want %v", observer.failures, want)
	}
}

func TestSearchFailsWhenEveryEngineFails(t *testing.T) {
	corpusTextFailure := errors.New("corpustext away")
	yacyFailure := errors.New("yacy away")
	observer := &recordingObserver{}
	searchStack := stack.NewStack([]stack.Engine{
		{Name: "corpustext", Engine: answeringEngine{failure: corpusTextFailure}},
		{Name: "yacy", Engine: answeringEngine{failure: yacyFailure}},
	}, observer)

	_, err := searchStack.SearchResultsFor(context.Background(), searchQuery)
	if !errors.Is(err, corpusTextFailure) || !errors.Is(err, yacyFailure) {
		t.Errorf("search = %v, want both failures", err)
	}
	if len(observer.failures) != 2 {
		t.Errorf("failures = %v, want both engines reported", observer.failures)
	}
}
