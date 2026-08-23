package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

type RecrawlRule interface {
	DecisionFor(
		ctx context.Context,
		canonicalURL canonicalurl.CanonicalURL,
	) (RecrawlDecision, error)
	RecordVisit(
		ctx context.Context,
		canonicalURL canonicalurl.CanonicalURL,
		version pagefetch.PageVersion,
	) error
}

type RecrawlDecision struct {
	Due     bool
	Version pagefetch.PageVersion
}
