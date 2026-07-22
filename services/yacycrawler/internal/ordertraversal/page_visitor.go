package ordertraversal

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisit"
)

type PageVisitor interface {
	Visit(ctx context.Context, url string) (pagevisit.VisitOutcome, error)
}
