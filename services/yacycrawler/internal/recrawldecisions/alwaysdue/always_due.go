package alwaysdue

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type AlwaysDue struct{}

func (AlwaysDue) Revisit(context.Context, string) (pagevisit.Revisit, error) {
	return pagevisit.Revisit{Due: true}, nil
}

func (AlwaysDue) Visited(context.Context, string, pagevisit.Revisit) error {
	return nil
}
