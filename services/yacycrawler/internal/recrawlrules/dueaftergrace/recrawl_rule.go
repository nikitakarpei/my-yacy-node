// Package dueaftergrace holds a page due for another visit once a configured
// grace window has elapsed since its last page visit, and supplies the page
// version that visit saw.
package dueaftergrace

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type Clock interface {
	Now() time.Time
}

type VisitedPages interface {
	LastPageVisitOf(
		ctx context.Context,
		canonicalURL canonicalurl.CanonicalURL,
	) (pagevisit.PageVisit, bool, error)
}

type RecrawlRule struct {
	visitedPages VisitedPages
	clock        Clock
	grace        time.Duration
}

func New(visitedPages VisitedPages, clock Clock, grace time.Duration) *RecrawlRule {
	return &RecrawlRule{visitedPages: visitedPages, clock: clock, grace: grace}
}

func (rule *RecrawlRule) RecrawlDecisionFor(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (pagevisit.RecrawlDecision, error) {
	lastVisit, visited, err := rule.visitedPages.LastPageVisitOf(ctx, canonicalURL)
	if err != nil {
		return pagevisit.RecrawlDecision{}, err
	}
	if !visited {
		return pagevisit.RecrawlDecision{Due: true}, nil
	}
	return pagevisit.RecrawlDecision{
		Due:     rule.clock.Now().Sub(lastVisit.VisitedAt) >= rule.grace,
		Version: lastVisit.Version,
	}, nil
}
