// Package dueaftergrace holds a page due for another visit once a configured
// grace window has elapsed since its last page visit.
package dueaftergrace

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type Clock interface {
	Now() time.Time
}

type RecrawlRule struct {
	clock Clock
	grace time.Duration
}

func New(clock Clock, grace time.Duration) *RecrawlRule {
	return &RecrawlRule{clock: clock, grace: grace}
}

func (rule *RecrawlRule) PageDueForRecrawl(lastVisit pagevisit.LastPageVisit) bool {
	return rule.clock.Now().Sub(lastVisit.VisitedAt) >= rule.grace
}
