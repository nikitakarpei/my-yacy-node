package pagevisit

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type Frontier interface {
	Add(tx *vault.Txn, pageVisits []pendingpagevisit.PendingPageVisit) error
}
