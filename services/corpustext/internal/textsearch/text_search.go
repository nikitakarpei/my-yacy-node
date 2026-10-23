// Package textsearch finds the indexed pages whose text holds every word of a query, and
// holds the ranking that every search index applies to them.
package textsearch

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

const (
	TitleWeight         = 6
	ContentWeight       = 2
	AuxiliaryTextWeight = 1
	SnippetLength       = 300
)

type Matcher interface {
	MatchesFor(ctx context.Context, search Search) ([]searchdocument.PageMatch, error)
}

type TextSearch struct {
	matcher  Matcher
	observer SearchObserver
}

func New(matcher Matcher, observer SearchObserver) *TextSearch {
	return &TextSearch{matcher: matcher, observer: observer}
}

func (s *TextSearch) MatchesFor(
	ctx context.Context,
	search Search,
) ([]searchdocument.PageMatch, bool) {
	startedAt := time.Now()
	matches, err := s.matcher.MatchesFor(ctx, search)
	elapsed := time.Since(startedAt)
	if err != nil {
		s.observer.SearchFailed(ctx, search, elapsed, err)
		return nil, false
	}
	presentableMatches := presentableMatchesAmong(matches)
	s.observer.SearchAnswered(ctx, search, elapsed, len(presentableMatches))
	return presentableMatches, true
}

func presentableMatchesAmong(matches []searchdocument.PageMatch) []searchdocument.PageMatch {
	presentableMatches := make([]searchdocument.PageMatch, 0, len(matches))
	for _, match := range matches {
		if match.URL == "" || match.Title == "" {
			continue
		}
		presentableMatches = append(presentableMatches, match)
	}
	return presentableMatches
}
