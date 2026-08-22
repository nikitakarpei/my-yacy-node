//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/searxngsearch"
)

func assertScrapedCorpusIsSearchable(t *testing.T, ctx context.Context, searxngBaseURL string) {
	t.Helper()
	assertScrapeRequestIsFound(t, ctx, searxngBaseURL)
	assertStemmedTermMatches(t, ctx, searxngBaseURL)
	assertSearchLanguageSelectsItsIndex(t, ctx, searxngBaseURL)
	assertCatchAllPageIsFound(t, ctx, searxngBaseURL)
	assertUnmatchedTermFindsNothing(t, ctx, searxngBaseURL)
}

func assertScrapeRequestIsFound(t *testing.T, ctx context.Context, searxngBaseURL string) {
	t.Helper()
	result := searxngsearch.OneResultInAnyLanguage(
		t, ctx, searxngBaseURL, "!"+engineBang+" "+englishSearchTerm,
	)
	if result.Title != englishTitle {
		t.Errorf("result title = %q, want %q", result.Title, englishTitle)
	}
	if result.URL != englishURL {
		t.Errorf("result url = %q, want %q", result.URL, englishURL)
	}
	if !strings.Contains(result.Content, englishSearchTerm) {
		t.Errorf("result content = %q, want it to carry %q", result.Content, englishSearchTerm)
	}
}

func assertStemmedTermMatches(t *testing.T, ctx context.Context, searxngBaseURL string) {
	t.Helper()
	result := searxngsearch.OneResultInAnyLanguage(
		t, ctx, searxngBaseURL, "!"+engineBang+" "+englishStemmed,
	)
	if result.URL != englishURL {
		t.Errorf("stemmed search url = %q, want %q", result.URL, englishURL)
	}
}

func assertSearchLanguageSelectsItsIndex(
	t *testing.T,
	ctx context.Context,
	searxngBaseURL string,
) {
	t.Helper()
	german := searxngsearch.ResultsInLanguage(
		t, ctx, searxngBaseURL, "!"+engineBang+" "+germanSearchTerm, germanLanguage,
	)
	if len(german) != 1 || german[0].URL != germanURL {
		t.Errorf("german search returned %v, want only %q", german, germanURL)
	}
	english := searxngsearch.ResultsInLanguage(
		t, ctx, searxngBaseURL, "!"+engineBang+" "+germanSearchTerm, englishLanguage,
	)
	if len(english) != 0 {
		t.Errorf("english search returned %v, want no result", english)
	}
}

func assertCatchAllPageIsFound(t *testing.T, ctx context.Context, searxngBaseURL string) {
	t.Helper()
	result := searxngsearch.OneResultInAnyLanguage(
		t, ctx, searxngBaseURL, "!"+engineBang+" "+spanishSearchTerm,
	)
	if result.URL != spanishURL {
		t.Errorf("catch-all search url = %q, want %q", result.URL, spanishURL)
	}
}

func assertUnmatchedTermFindsNothing(t *testing.T, ctx context.Context, searxngBaseURL string) {
	t.Helper()
	results := searxngsearch.ResultsInAnyLanguage(
		t, ctx, searxngBaseURL, "!"+engineBang+" nonexistentterm",
	)
	if len(results) != 0 {
		t.Errorf("no-match search returned %d results, want 0", len(results))
	}
}
