package alwaysdue

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type AlwaysDue struct{}

func (AlwaysDue) DecisionFor(context.Context, string) (pagevisit.RecrawlDecision, error) {
	return pagevisit.RecrawlDecision{Due: true}, nil
}

func (AlwaysDue) RecordVisit(context.Context, string, pagefetch.PageVersion) error {
	return nil
}
