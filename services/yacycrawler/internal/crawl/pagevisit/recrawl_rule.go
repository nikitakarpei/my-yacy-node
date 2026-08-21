package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
)

type RecrawlRule interface {
	DecisionFor(ctx context.Context, canonicalURL string) (RecrawlDecision, error)
	RecordVisit(ctx context.Context, canonicalURL string, version pagefetch.PageVersion) error
}

type RecrawlDecision struct {
	Due     bool
	Version pagefetch.PageVersion
}
