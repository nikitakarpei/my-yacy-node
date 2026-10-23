package crawlprogress

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	completedVisitsOfCrawlBucket = "crawl_completed_page_visits"
	completedVisitsOfHostBucket  = "crawl_host_completed_page_visits"
	droppedVisitsOfCrawlBucket   = "crawl_dropped_page_visits"
	droppedVisitsOfHostBucket    = "crawl_host_dropped_page_visits"
	unadmittedLinksBucket        = "crawl_unadmitted_links"
)

func registerCounts(storage *vault.Vault) (*Counts, error) {
	completed, err := registerVisitCounts[yacycrawlcontract.PageVisitResult](
		storage, completedVisitsOfCrawlBucket, completedVisitsOfHostBucket,
	)
	if err != nil {
		return nil, err
	}
	dropped, err := registerVisitCounts[yacycrawlcontract.PageVisitDropReason](
		storage, droppedVisitsOfCrawlBucket, droppedVisitsOfHostBucket,
	)
	if err != nil {
		return nil, err
	}
	unadmittedLinks, err := storage.RegisterAmounts(unadmittedLinksBucket, crawlKeys.KeyLayout())
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", unadmittedLinksBucket, err)
	}
	return &Counts{completed: completed, dropped: dropped, unadmittedLinks: unadmittedLinks}, nil
}
