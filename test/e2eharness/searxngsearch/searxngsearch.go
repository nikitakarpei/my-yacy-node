//go:build e2e

// Package searxngsearch queries a running SearXNG instance's JSON search API.
package searxngsearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

type Result struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type response struct {
	Results []Result `json:"results"`
}

func ResultsInAnyLanguage(t *testing.T, ctx context.Context, baseURL, query string) []Result {
	t.Helper()
	return results(t, ctx, baseURL, url.Values{"q": {query}, "format": {"json"}})
}

func ResultsInLanguage(
	t *testing.T,
	ctx context.Context,
	baseURL, query, language string,
) []Result {
	t.Helper()
	return results(t, ctx, baseURL, url.Values{
		"q":        {query},
		"format":   {"json"},
		"language": {language},
	})
}

func OneResultInAnyLanguage(t *testing.T, ctx context.Context, baseURL, query string) Result {
	t.Helper()
	found := ResultsInAnyLanguage(t, ctx, baseURL, query)
	if len(found) == 0 {
		t.Fatal("search response carries no results")
	}
	return found[0]
}

func results(t *testing.T, ctx context.Context, baseURL string, values url.Values) []Result {
	t.Helper()
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, baseURL+"/search?"+values.Encode(), nil,
	)
	if err != nil {
		t.Fatalf("build search request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("search request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search request: status %d", resp.StatusCode)
	}

	var decoded response
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	return decoded.Results
}
