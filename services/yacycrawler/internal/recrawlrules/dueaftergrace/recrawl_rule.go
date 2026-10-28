// Package dueaftergrace holds a page due for another visit once the recrawl grace
// of its order has elapsed since its last page visit. The visit then asks the
// origin whether the page changed; with no grace, it fetches the page in full.
package dueaftergrace

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type Clock interface {
	Now() time.Time
}

type RecrawlRule struct {
	clock Clock
}

func New(clock Clock) *RecrawlRule {
	return &RecrawlRule{clock: clock}
}

func (rule *RecrawlRule) RecrawlOf(
	lastVisit pagevisit.LastPageVisit,
	recrawlGrace yacycrawlcontract.RecrawlGrace,
) pagevisit.Recrawl {
	switch {
	case recrawlGrace == yacycrawlcontract.NoRecrawlGrace:
		return pagevisit.Recrawl{Due: true}
	case recrawlGrace.Covers(rule.clock.Now().Sub(lastVisit.VisitedAt)):
		return pagevisit.Recrawl{Due: false}
	default:
		return pagevisit.Recrawl{Due: true, KnownVersion: lastVisit.Version}
	}
}
