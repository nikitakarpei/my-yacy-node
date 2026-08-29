//go:build e2e

package e2e

import (
	"context"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

const (
	searchQuery        = "!" + testEngineBang + " research subject"
	askedForResults    = 1
	firstReadLimit     = 40
	secondReadLimit    = 400
	markdownFragment   = "research subject"
	routedLinkFragment = "/visit"
)

func TestServiceServesASearchToolAndAPageTool(t *testing.T) {
	ctx := context.Background()

	session := openToolSession(t, ctx, startWebResearchStack(t, ctx))

	served := toolNamesOf(t, ctx, session)
	for _, wanted := range []string{searchWebToolName, readPageToolName} {
		if !slices.Contains(served, wanted) {
			t.Errorf("served tools = %v, want them to carry %q", served, wanted)
		}
	}
}

func TestSearchAnswersWithTheDestinationLinkOfEachResult(t *testing.T) {
	ctx := context.Background()

	session := openToolSession(t, ctx, startWebResearchStack(t, ctx))

	results := searchResultsFor(t, ctx, session, searchCall{Query: searchQuery})
	if len(results) != engineResults {
		t.Fatalf("search results = %v, want the %d the engine answers", results, engineResults)
	}
	for _, result := range results {
		if strings.Contains(result.URL, routedLinkFragment) {
			t.Errorf("result url = %q, want a link SearXNG left as it is", result.URL)
		}
	}
	if results[0].URL != originCanonicalURL {
		t.Errorf("first result url = %q, want %q", results[0].URL, originCanonicalURL)
	}
	if results[0].Title != resultTitle {
		t.Errorf("first result title = %q, want %q", results[0].Title, resultTitle)
	}
	if results[1].URL != secondResultURL || results[2].URL != thirdResultURL {
		t.Errorf("result urls = %v, want them in the order SearXNG returns them", results)
	}
}

func TestSearchCarriesAtMostTheNumberOfResultsItAsksFor(t *testing.T) {
	ctx := context.Background()

	session := openToolSession(t, ctx, startWebResearchStack(t, ctx))

	results := searchResultsFor(t, ctx, session, searchCall{
		Query:       searchQuery,
		ResultLimit: askedForResults,
	})
	if len(results) != askedForResults {
		t.Fatalf("search results = %v, want the %d asked for", results, askedForResults)
	}
	if results[0].URL != originCanonicalURL {
		t.Errorf(
			"result url = %q, want the first SearXNG returns, %q",
			results[0].URL, originCanonicalURL,
		)
	}
}

func TestPageCallFetchesThePageAndAnswersWithItsMarkdown(t *testing.T) {
	ctx := context.Background()

	session := openToolSession(t, ctx, startWebResearchStack(t, ctx))

	answer := pageAnswerFor(t, ctx, session, pageCall{URL: originCanonicalURL})
	if answer.URL != originCanonicalURL {
		t.Errorf("page url = %q, want %q", answer.URL, originCanonicalURL)
	}
	if answer.FetchOutcome != pageFetched {
		t.Errorf("fetch outcome = %q, want %q", answer.FetchOutcome, pageFetched)
	}
	if !strings.Contains(answer.Markdown, markdownFragment) {
		t.Errorf("page markdown = %q, want it to contain %q", answer.Markdown, markdownFragment)
	}
	if answer.Version == "" {
		t.Error("page answer carries no version")
	}
	if answer.StoredAt.IsZero() {
		t.Error("page answer carries no time the corpus stored the version")
	}
	if answer.Truncated {
		t.Error("page answer says it is truncated, want the whole markdown")
	}
	if answer.MarkdownCharacterCount != utf8.RuneCountInString(answer.Markdown) {
		t.Errorf(
			"markdown character count = %d, want the %d characters carried",
			answer.MarkdownCharacterCount, utf8.RuneCountInString(answer.Markdown),
		)
	}
}

func TestPageCallNamingAVersionAnswersThatVersionWithoutFetchingAgain(t *testing.T) {
	ctx := context.Background()

	session := openToolSession(t, ctx, startWebResearchStack(t, ctx))

	firstAnswer := pageAnswerFor(t, ctx, session, pageCall{
		URL:            originCanonicalURL,
		CharacterLimit: firstReadLimit,
	})
	if !firstAnswer.Truncated {
		t.Fatalf("page answer of %d characters is not truncated, want it truncated", firstReadLimit)
	}
	if utf8.RuneCountInString(firstAnswer.Markdown) != firstReadLimit {
		t.Fatalf(
			"page answer carries %d characters, want the %d asked for",
			utf8.RuneCountInString(firstAnswer.Markdown), firstReadLimit,
		)
	}
	if firstAnswer.MarkdownCharacterCount <= firstReadLimit {
		t.Fatalf(
			"whole markdown character count = %d, want more than the %d carried",
			firstAnswer.MarkdownCharacterCount, firstReadLimit,
		)
	}

	secondAnswer := pageAnswerFor(t, ctx, session, pageCall{
		URL:            originCanonicalURL,
		CharacterLimit: secondReadLimit,
		Version:        firstAnswer.Version,
	})
	if secondAnswer.Version != firstAnswer.Version {
		t.Errorf("page version = %q, want %q", secondAnswer.Version, firstAnswer.Version)
	}
	if secondAnswer.FetchOutcome != fetchNotNeeded {
		t.Errorf("fetch outcome = %q, want %q", secondAnswer.FetchOutcome, fetchNotNeeded)
	}
	if !strings.HasPrefix(secondAnswer.Markdown, firstAnswer.Markdown) {
		t.Errorf(
			"page markdown = %q, want it to continue %q",
			secondAnswer.Markdown, firstAnswer.Markdown,
		)
	}
}

func TestPageCallWithinTheToleratedAgeAnswersWithoutFetchingAgain(t *testing.T) {
	ctx := context.Background()

	session := openToolSession(t, ctx, startWebResearchStack(t, ctx))

	firstAnswer := pageAnswerFor(t, ctx, session, pageCall{URL: originCanonicalURL})
	if firstAnswer.FetchOutcome != pageFetched {
		t.Fatalf(
			"fetch outcome = %q, want %q",
			firstAnswer.FetchOutcome, pageFetched,
		)
	}

	secondAnswer := pageAnswerFor(t, ctx, session, pageCall{URL: originCanonicalURL})
	if secondAnswer.FetchOutcome != fetchNotNeeded {
		t.Errorf(
			"fetch outcome = %q, want %q for markdown the corpus stored moments ago",
			secondAnswer.FetchOutcome, fetchNotNeeded,
		)
	}
	if secondAnswer.Version != firstAnswer.Version {
		t.Errorf("page version = %q, want %q", secondAnswer.Version, firstAnswer.Version)
	}
	if !secondAnswer.StoredAt.Equal(firstAnswer.StoredAt) {
		t.Errorf("stored at = %v, want %v", secondAnswer.StoredAt, firstAnswer.StoredAt)
	}
}
