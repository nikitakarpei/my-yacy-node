package pagevisitintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type reportingPageVisitQueue struct {
	pendingPageVisits PendingPageVisits
	observer          PendingPageVisitObserver
}

func (queue reportingPageVisitQueue) NextUpTo(
	ctx context.Context,
	amount int,
) ([]pendingpagevisit.LeasedPageVisit, error) {
	leasedPageVisits, err := queue.pendingPageVisits.Lease(ctx, amount)
	if err != nil {
		return nil, err
	}
	queue.observer.PendingPageVisitsLeased(ctx, len(leasedPageVisits))
	return leasedPageVisits, nil
}
