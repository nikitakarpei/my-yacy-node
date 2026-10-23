// Package stack searches every search engine of the operator's stack at once, and answers
// with their results interleaved by rank: the first result of each engine, then the second
// of each, and so on, each page once. An engine that fails is left out of the answer; the
// search fails only when every engine fails.
package stack

import (
	"context"
	"errors"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/websearch"
)

type Engine struct {
	Name   string
	Engine websearch.SearchEngine
}

type Stack struct {
	engines  []Engine
	observer EngineObserver
}

func NewStack(engines []Engine, observer EngineObserver) *Stack {
	return &Stack{engines: engines, observer: observer}
}

func (s *Stack) SearchResultsFor(
	ctx context.Context,
	query string,
) ([]websearch.SearchResult, error) {
	answers := s.answersFor(ctx, query)
	rankedResults, failures := s.rankedResultsFrom(ctx, query, answers)
	if len(rankedResults) == 0 {
		return nil, failures
	}
	return interleavedFrom(rankedResults), nil
}

type engineAnswer struct {
	searchResults []websearch.SearchResult
	failure       error
}

func (s *Stack) answersFor(ctx context.Context, query string) []engineAnswer {
	answers := make([]engineAnswer, len(s.engines))
	var searches sync.WaitGroup
	for index, engine := range s.engines {
		searches.Go(func() {
			searchResults, err := engine.Engine.SearchResultsFor(ctx, query)
			answers[index] = engineAnswer{searchResults: searchResults, failure: err}
		})
	}
	searches.Wait()
	return answers
}

func (s *Stack) rankedResultsFrom(
	ctx context.Context,
	query string,
	answers []engineAnswer,
) ([][]websearch.SearchResult, error) {
	var rankedResults [][]websearch.SearchResult
	var failures error
	for index, answer := range answers {
		if answer.failure != nil {
			s.observer.EngineFailed(ctx, s.engines[index].Name, query, answer.failure)
			failures = errors.Join(failures, answer.failure)
			continue
		}
		rankedResults = append(rankedResults, answer.searchResults)
	}
	return rankedResults, failures
}

func interleavedFrom(rankedResults [][]websearch.SearchResult) []websearch.SearchResult {
	var interleaved []websearch.SearchResult
	seenPages := map[string]bool{}
	for rank := 0; ; rank++ {
		rankReached := false
		for _, searchResults := range rankedResults {
			if rank >= len(searchResults) {
				continue
			}
			rankReached = true
			page := pageOf(searchResults[rank])
			if seenPages[page] {
				continue
			}
			seenPages[page] = true
			interleaved = append(interleaved, searchResults[rank])
		}
		if !rankReached {
			return interleaved
		}
	}
}

func pageOf(searchResult websearch.SearchResult) string {
	canonicalURL, err := canonicalurl.CanonicalURLOf(searchResult.URL)
	if err != nil {
		return searchResult.URL
	}
	return canonicalURL.String()
}
