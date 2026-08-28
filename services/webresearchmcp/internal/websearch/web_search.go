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
	Progress          SearchProgress
	SearchResultLimit int
}

type WebSearch struct {
	engine            SearchEngine
	progress          SearchProgress
	searchResultLimit int
}

func NewWebSearch(cfg Config) *WebSearch {
	return &WebSearch{
		engine:            cfg.Engine,
		progress:          cfg.Progress,
		searchResultLimit: cfg.SearchResultLimit,
	}
}

func (s *WebSearch) SearchResultsFor(
	ctx context.Context,
	query string,
	searchResultLimit int,
) ([]SearchResult, error) {
	everyResult, err := s.engine.SearchResultsFor(ctx, query)
	if err != nil {
		s.progress.SearchFailed(ctx, query, err)
		return nil, fmt.Errorf("search for %q: %w", query, err)
	}
	carriedResults := s.carriedResultsFrom(everyResult, searchResultLimit)
	s.progress.SearchServed(ctx, query, len(carriedResults))
	return carriedResults, nil
}

func (s *WebSearch) carriedResultsFrom(
	everyResult []SearchResult,
	searchResultLimit int,
) []SearchResult {
	carriedResultLimit := searchResultLimit
	if carriedResultLimit <= 0 {
		carriedResultLimit = s.searchResultLimit
	}
	if len(everyResult) <= carriedResultLimit {
		return everyResult
	}
	return everyResult[:carriedResultLimit]
}
