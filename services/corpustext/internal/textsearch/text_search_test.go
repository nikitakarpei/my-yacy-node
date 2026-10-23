package textsearch_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

type answeringMatcher struct {
	matches []searchdocument.PageMatch
	err     error
}

func (index answeringMatcher) MatchesFor(
	_ context.Context,
	_ textsearch.Search,
) ([]searchdocument.PageMatch, error) {
	return index.matches, index.err
}

type recordingObserver struct {
	answeredAmounts []int
	failures        []error
}

func (o *recordingObserver) SearchAnswered(
	_ context.Context,
	_ textsearch.Search,
	_ time.Duration,
	amountOfMatches int,
) {
	o.answeredAmounts = append(o.answeredAmounts, amountOfMatches)
}

func (o *recordingObserver) SearchFailed(
	_ context.Context,
	_ textsearch.Search,
	_ time.Duration,
	cause error,
) {
	o.failures = append(o.failures, cause)
}

func wordsSearch(t *testing.T) textsearch.Search {
	t.Helper()
	search, err := textsearch.SearchFrom("words", "", 0, textsearch.DefaultMatchLimit)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	return search
}

func TestTextSearchYieldsTheMatchesThatCarryAURLAndATitle(t *testing.T) {
	presentable := searchdocument.PageMatch{
		URL: "https://example.com/", Title: "Hi", Snippet: "words here",
	}
	index := answeringMatcher{matches: []searchdocument.PageMatch{
		presentable,
		{URL: "https://example.com/untitled", Snippet: "words"},
		{Title: "Nowhere", Snippet: "words"},
	}}
	observer := &recordingObserver{}

	matches, answered := textsearch.New(index, textsearch.SearchObservers{observer}).
		MatchesFor(context.Background(), wordsSearch(t))

	if !answered {
		t.Fatal("search not answered")
	}
	if !slices.Equal(matches, []searchdocument.PageMatch{presentable}) {
		t.Errorf("matches = %+v", matches)
	}
	if !slices.Equal(observer.answeredAmounts, []int{1}) {
		t.Errorf("answered amounts = %v", observer.answeredAmounts)
	}
}

func TestTextSearchReportsAFailedMatcher(t *testing.T) {
	matcherFailure := errors.New("index down")
	observer := &recordingObserver{}

	matches, answered := textsearch.New(answeringMatcher{err: matcherFailure}, observer).
		MatchesFor(context.Background(), wordsSearch(t))

	if answered || matches != nil {
		t.Errorf("matches = %+v, answered = %v", matches, answered)
	}
	if len(observer.failures) != 1 || !errors.Is(observer.failures[0], matcherFailure) {
		t.Errorf("failures = %v", observer.failures)
	}
}
