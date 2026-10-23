package pagevisit

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type Frontier interface {
	AddNextOf(
		tx *vault.Txn,
		pageVisit pendingpagevisit.PendingPageVisit,
		nextPageVisits []pendingpagevisit.PendingPageVisit,
	) error
}
