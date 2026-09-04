// Package websearch answers a query with the results the operator's own search engine
// returns, in the order the engine returns them, cut to the number of results the caller
// wants. It ranks nothing, filters nothing, and keeps no history of what it answered.
package websearch

import (
	"context"
	"fmt"
)

type SearchResult struct {
	URL     string
	Title   string
	Snippet string
}

type SearchEngine interface {
	SearchResultsFor(ctx context.Context, query string) ([]SearchResult, error)
}

type Config struct {
	Engine            SearchEngine
	Observer          WebSearchObserver
	SearchResultLimit int
}

type WebSearch struct {
	engine            SearchEngine
	observer          WebSearchObserver
	searchResultLimit int
}

func NewWebSearch(cfg Config) *WebSearch {
	return &WebSearch{
		engine:            cfg.Engine,
		observer:          cfg.Observer,
		searchResultLimit: cfg.SearchResultLimit,
	}
}

func (s *WebSearch) SearchResultsFor(
	ctx context.Context,
	query string,
	requestedSearchResultLimit int,
) ([]SearchResult, error) {
	engineSearchResults, err := s.engine.SearchResultsFor(ctx, query)
	if err != nil {
		s.observer.SearchFailed(ctx, query, err)
		return nil, fmt.Errorf("search for %q: %w", query, err)
	}
	searchResults := s.searchResultsWithinLimitFrom(
		engineSearchResults,
		requestedSearchResultLimit,
	)
	s.observer.SearchServed(ctx, query, len(searchResults))
	return searchResults, nil
}

func (s *WebSearch) searchResultsWithinLimitFrom(
	engineSearchResults []SearchResult,
	requestedSearchResultLimit int,
) []SearchResult {
	searchResultLimit := requestedSearchResultLimit
	if searchResultLimit <= 0 {
		searchResultLimit = s.searchResultLimit
	}
	if len(engineSearchResults) <= searchResultLimit {
		return engineSearchResults
	}
	return engineSearchResults[:searchResultLimit]
}
