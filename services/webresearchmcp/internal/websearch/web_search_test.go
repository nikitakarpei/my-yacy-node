package websearch_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/websearch"
)

const configuredLimit = 3

var errEngineUnreachable = errors.New("engine unreachable")

type engineAnswering struct {
	results []websearch.SearchResult
	failure error
}

func (e engineAnswering) SearchResultsFor(
	_ context.Context,
	_ string,
) ([]websearch.SearchResult, error) {
	return e.results, e.failure
}

type recordingSearchProgress struct {
	servedResultCount int
	failures          int
}

func (p *recordingSearchProgress) SearchServed(_ context.Context, _ string, count int) {
	p.servedResultCount = count
}

func (p *recordingSearchProgress) SearchFailed(_ context.Context, _ string, _ error) {
	p.failures++
}

func TestSearchAnswersWithTheResultsInTheOrderTheEngineReturnsThem(t *testing.T) {
	search := websearch.NewWebSearch(websearch.Config{
		Engine:            engineAnswering{results: resultsNumbered(3)},
		Progress:          &recordingSearchProgress{},
		SearchResultLimit: configuredLimit,
	})

	results, err := search.SearchResultsFor(context.Background(), "a query", 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results = %v, want the three the engine returns", results)
	}
	for position, result := range results {
		if result.URL != urlNumbered(position) {
			t.Errorf("result %d url = %q, want %q", position, result.URL, urlNumbered(position))
		}
	}
}

func TestSearchCarriesAtMostTheConfiguredNumberOfResults(t *testing.T) {
	search := websearch.NewWebSearch(websearch.Config{
		Engine:            engineAnswering{results: resultsNumbered(10)},
		Progress:          &recordingSearchProgress{},
		SearchResultLimit: configuredLimit,
	})

	results, err := search.SearchResultsFor(context.Background(), "a query", 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != configuredLimit {
		t.Fatalf("results = %d, want the configured %d", len(results), configuredLimit)
	}
	if results[0].URL != urlNumbered(0) {
		t.Errorf("first result url = %q, want the first the engine returns", results[0].URL)
	}
}

func TestSearchCarriesTheNumberOfResultsTheCallerNames(t *testing.T) {
	search := websearch.NewWebSearch(websearch.Config{
		Engine:            engineAnswering{results: resultsNumbered(10)},
		Progress:          &recordingSearchProgress{},
		SearchResultLimit: configuredLimit,
	})

	results, err := search.SearchResultsFor(context.Background(), "a query", 1)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want the one the caller names", len(results))
	}
}

func TestSearchTellsTheProgressObserverHowManyResultsItServed(t *testing.T) {
	progress := &recordingSearchProgress{}
	search := websearch.NewWebSearch(websearch.Config{
		Engine:            engineAnswering{results: resultsNumbered(10)},
		Progress:          progress,
		SearchResultLimit: configuredLimit,
	})

	if _, err := search.SearchResultsFor(context.Background(), "a query", 0); err != nil {
		t.Fatalf("search: %v", err)
	}
	if progress.servedResultCount != configuredLimit {
		t.Errorf("served result count = %d, want %d", progress.servedResultCount, configuredLimit)
	}
}

func TestSearchPassesTheEngineFailureOnAndReportsIt(t *testing.T) {
	progress := &recordingSearchProgress{}
	search := websearch.NewWebSearch(websearch.Config{
		Engine:            engineAnswering{failure: errEngineUnreachable},
		Progress:          progress,
		SearchResultLimit: configuredLimit,
	})

	_, err := search.SearchResultsFor(context.Background(), "a query", 0)
	if !errors.Is(err, errEngineUnreachable) {
		t.Fatalf("search error = %v, want the engine failure", err)
	}
	if progress.failures != 1 {
		t.Errorf("reported failures = %d, want 1", progress.failures)
	}
}

func resultsNumbered(count int) []websearch.SearchResult {
	results := make([]websearch.SearchResult, 0, count)
	for position := range count {
		results = append(results, websearch.SearchResult{
			URL:     urlNumbered(position),
			Title:   "Title " + strconv.Itoa(position),
			Snippet: "Snippet " + strconv.Itoa(position),
		})
	}
	return results
}

func urlNumbered(position int) string {
	return "https://example.org/" + strconv.Itoa(position)
}
