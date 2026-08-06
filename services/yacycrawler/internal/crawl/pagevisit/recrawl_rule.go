package pagevisit

import (
	"context"
	"time"
)

type RecrawlRule interface {
	DecisionFor(ctx context.Context, canonicalURL string) (RecrawlDecision, error)
	RecordVisit(ctx context.Context, canonicalURL string, version PageVersion) error
}

type RecrawlDecision struct {
	Due     bool
	Version PageVersion
}

type PageVersion struct {
	EntityTag  string
	ModifiedAt time.Time
}
