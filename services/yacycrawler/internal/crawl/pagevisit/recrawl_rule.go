package pagevisit

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type RecrawlRule interface {
	DecisionFor(
		ctx context.Context,
		canonicalURL yacycrawlcontract.CanonicalURL,
	) (RecrawlDecision, error)
	RecordVisit(
		ctx context.Context,
		canonicalURL yacycrawlcontract.CanonicalURL,
		version PageVersion,
	) error
}

type RecrawlDecision struct {
	Due     bool
	Version PageVersion
}

type PageVersion struct {
	EntityTag  string
	ModifiedAt time.Time
}
