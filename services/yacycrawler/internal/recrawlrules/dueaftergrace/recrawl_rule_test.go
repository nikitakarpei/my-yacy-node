package dueaftergrace_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
)

var now = time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type lastPageVisit struct {
	visit   pagevisit.PageVisit
	visited bool
	err     error
}

func (pages lastPageVisit) LastPageVisitOf(
	context.Context,
	canonicalurl.CanonicalURL,
) (pagevisit.PageVisit, bool, error) {
	return pages.visit, pages.visited, pages.err
}

func decisionOf(t *testing.T, pages dueaftergrace.VisitedPages) pagevisit.RecrawlDecision {
	t.Helper()
	decision, err := dueaftergrace.New(pages, fixedClock{now: now}, time.Hour).RecrawlDecisionFor(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
	)
	if err != nil {
		t.Fatalf("recrawl decision: %v", err)
	}
	return decision
}

func TestAPageNeverVisitedIsDueWithNoPageVersion(t *testing.T) {
	decision := decisionOf(t, lastPageVisit{})

	if !decision.Due || decision.Version != (pagefetch.PageVersion{}) {
		t.Fatalf("decision = %+v, want due with no page version", decision)
	}
}

func TestAPageVisitedWithinTheGraceIsNotDueButKeepsItsPageVersion(t *testing.T) {
	version := pagefetch.PageVersion{EntityTag: `"abc"`}

	decision := decisionOf(t, lastPageVisit{
		visit:   pagevisit.PageVisit{VisitedAt: now.Add(-30 * time.Minute), Version: version},
		visited: true,
	})

	if decision.Due {
		t.Fatal("want not due within the grace window")
	}
	if decision.Version != version {
		t.Fatalf("page version = %+v, want %+v", decision.Version, version)
	}
}

func TestAPageVisitedBeforeTheGraceIsDueAndKeepsItsPageVersion(t *testing.T) {
	version := pagefetch.PageVersion{EntityTag: `"abc"`}

	decision := decisionOf(t, lastPageVisit{
		visit:   pagevisit.PageVisit{VisitedAt: now.Add(-2 * time.Hour), Version: version},
		visited: true,
	})

	if !decision.Due {
		t.Fatal("want due outside the grace window")
	}
	if decision.Version != version {
		t.Fatalf("page version = %+v, want %+v", decision.Version, version)
	}
}

func TestAnUnreadableLastPageVisitFailsFast(t *testing.T) {
	boom := errors.New("boom")

	_, err := dueaftergrace.New(
		lastPageVisit{err: boom}, fixedClock{now: now}, time.Hour,
	).RecrawlDecisionFor(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
	)

	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want the failure of the visited pages", err)
	}
}
