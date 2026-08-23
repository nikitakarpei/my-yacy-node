package alwaysdue

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type AlwaysDue struct{}

func (AlwaysDue) DecisionFor(
	context.Context,
	canonicalurl.CanonicalURL,
) (pagevisit.RecrawlDecision, error) {
	return pagevisit.RecrawlDecision{Due: true}, nil
}

func (AlwaysDue) RecordVisit(
	context.Context,
	canonicalurl.CanonicalURL,
	pagefetch.PageVersion,
) error {
	return nil
}
