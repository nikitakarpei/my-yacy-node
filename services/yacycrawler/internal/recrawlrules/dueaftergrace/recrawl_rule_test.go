package dueaftergrace_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
)

var (
	now          = time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	knownVersion = pagefetch.PageVersion{EntityTag: `"known"`}
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func recrawlOf(visitedAt time.Time, recrawlGrace time.Duration) pagevisit.Recrawl {
	return dueaftergrace.New(fixedClock{now: now}).RecrawlOf(
		pagevisit.LastPageVisit{VisitedAt: visitedAt, Version: knownVersion},
		recrawlGrace,
	)
}

func TestAPageVisitedWithinTheGraceIsNotDue(t *testing.T) {
	if recrawlOf(now.Add(-30*time.Minute), time.Hour).Due {
		t.Fatal("want not due within the grace window")
	}
}

func TestAPageVisitedBeforeTheGraceIsDueAtItsKnownVersion(t *testing.T) {
	recrawl := recrawlOf(now.Add(-2*time.Hour), time.Hour)
	if !recrawl.Due || recrawl.KnownVersion != knownVersion {
		t.Fatalf("recrawl = %+v, want due at the known version", recrawl)
	}
}

func TestAPageVisitedExactlyOneGraceAgoIsDue(t *testing.T) {
	if !recrawlOf(now.Add(-time.Hour), time.Hour).Due {
		t.Fatal("want due once the grace window has elapsed")
	}
}

func TestWithNoGraceAPageJustVisitedIsDueInFull(t *testing.T) {
	recrawl := recrawlOf(now.Add(time.Minute), yacycrawlcontract.NoRecrawlGrace)
	if !recrawl.Due || recrawl.KnownVersion != (pagefetch.PageVersion{}) {
		t.Fatalf("recrawl = %+v, want due with no known version", recrawl)
	}
}
