package dueaftergrace_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
)

var now = time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func dueForRecrawl(visitedAt time.Time) bool {
	return dueaftergrace.New(fixedClock{now: now}, time.Hour).
		PageDueForRecrawl(pagevisit.PageVisit{VisitedAt: visitedAt})
}

func TestAPageVisitedWithinTheGraceIsNotDue(t *testing.T) {
	if dueForRecrawl(now.Add(-30 * time.Minute)) {
		t.Fatal("want not due within the grace window")
	}
}

func TestAPageVisitedBeforeTheGraceIsDue(t *testing.T) {
	if !dueForRecrawl(now.Add(-2 * time.Hour)) {
		t.Fatal("want due outside the grace window")
	}
}

func TestAPageVisitedExactlyOneGraceAgoIsDue(t *testing.T) {
	if !dueForRecrawl(now.Add(-time.Hour)) {
		t.Fatal("want due once the grace window has elapsed")
	}
}
