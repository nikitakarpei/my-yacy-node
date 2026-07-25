package pagevisit

import (
	"context"
	"time"
)

type Revisit struct {
	Due        bool
	EntityTag  string
	ModifiedAt time.Time
}

type RecrawlDecision interface {
	Revisit(ctx context.Context, canonicalURL string) (Revisit, error)
	Visited(ctx context.Context, canonicalURL string, validators Revisit) error
}
