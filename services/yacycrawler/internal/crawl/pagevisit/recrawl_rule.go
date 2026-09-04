package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

type RecrawlRule interface {
	RecrawlDecisionFor(
		ctx context.Context,
		canonicalURL canonicalurl.CanonicalURL,
	) (RecrawlDecision, error)
}

type RecrawlDecision struct {
	Due     bool
	Version pagefetch.PageVersion
}
