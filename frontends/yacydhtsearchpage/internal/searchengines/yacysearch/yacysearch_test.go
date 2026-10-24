package yacysearch_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/searchengines/yacysearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/searchpages"
)

const yacyAnswer = `{"channels":[{"title":"YaCy Search Engine: DHT search","items":[
	{"title":"YaCy","link":"https://yacy.net/","description":"Decentralized web search",
	 "pubDate":"Wed, 21 Oct 2026 09:30:00 +0000"},
	{"title":"Undated","link":"https://example.org/","description":"","pubDate":""}]}]}`

func searchOver(t *testing.T, handler http.HandlerFunc) *yacysearch.YaCySearch {
	t.Helper()
	service := httptest.NewServer(handler)
	t.Cleanup(service.Close)
	serviceURL, err := url.Parse(service.URL + "/")
	if err != nil {
		t.Fatalf("parse %q: %v", service.URL, err)
	}
	return yacysearch.New(serviceURL, 5*time.Second)
}

func TestSearchResultsForAsksTheYaCySearchInterface(t *testing.T) {
	asked := make(chan *http.Request, 1)
	search := searchOver(t, func(w http.ResponseWriter, r *http.Request) {
		asked <- r
		_, _ = io.WriteString(w, yacyAnswer)
	})

	if _, err := search.SearchResultsFor(
		context.Background(),
		"decentral search",
		20,
		10,
	); err != nil {
		t.Fatalf("SearchResultsFor: %v", err)
	}

	request := <-asked
	if request.URL.Path != "/yacysearch.json" {
		t.Errorf("path = %q, want /yacysearch.json", request.URL.Path)
	}
	want := url.Values{
		"query": {"decentral search"}, "startRecord": {"20"}, "maximumRecords": {"10"},
	}
	if request.URL.Query().Encode() != want.Encode() {
		t.Errorf("query = %q, want %q", request.URL.RawQuery, want.Encode())
	}
}

func TestSearchResultsForReadsEveryResultOfTheAnswer(t *testing.T) {
	search := searchOver(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, yacyAnswer)
	})

	results, err := search.SearchResultsFor(context.Background(), "yacy", 0, 10)
	if err != nil {
		t.Fatalf("SearchResultsFor: %v", err)
	}

	want := []searchpages.SearchResult{
		{
			Title:       "YaCy",
			Address:     "https://yacy.net/",
			Description: "Decentralized web search",
			PublishedAt: time.Date(2026, 10, 21, 9, 30, 0, 0, time.UTC),
		},
		{Title: "Undated", Address: "https://example.org/"},
	}
	if len(results) != len(want) {
		t.Fatalf("results = %+v, want %+v", results, want)
	}
	for index := range want {
		if results[index].Title != want[index].Title ||
			results[index].Address != want[index].Address ||
			results[index].Description != want[index].Description ||
			!results[index].PublishedAt.Equal(want[index].PublishedAt) {
			t.Errorf("result %d = %+v, want %+v", index, results[index], want[index])
		}
	}
}

func TestSearchResultsForFailsWhenTheServiceRefuses(t *testing.T) {
	search := searchOver(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	if _, err := search.SearchResultsFor(context.Background(), "yacy", 0, 10); err == nil {
		t.Error("SearchResultsFor succeeded on a refused search")
	}
}

func TestSearchResultsForFailsOnAnUnreadableAnswer(t *testing.T) {
	search := searchOver(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "<html>")
	})

	if _, err := search.SearchResultsFor(context.Background(), "yacy", 0, 10); err == nil {
		t.Error("SearchResultsFor succeeded on an unreadable answer")
	}
}

func TestSearchResultsForFailsWhenTheServiceIsAway(t *testing.T) {
	away, err := url.Parse("http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if _, err := yacysearch.New(away, time.Second).SearchResultsFor(
		context.Background(), "yacy", 0, 10,
	); err == nil {
		t.Error("SearchResultsFor succeeded without a service")
	}
}
