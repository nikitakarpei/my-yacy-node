package alwaysdue

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type AlwaysDue struct{}

func (AlwaysDue) DecisionFor(
	context.Context,
	yacycrawlcontract.CanonicalURL,
) (pagevisit.RecrawlDecision, error) {
	return pagevisit.RecrawlDecision{Due: true}, nil
}

func (AlwaysDue) RecordVisit(
	context.Context,
	yacycrawlcontract.CanonicalURL,
	pagevisit.PageVersion,
) error {
	return nil
}
