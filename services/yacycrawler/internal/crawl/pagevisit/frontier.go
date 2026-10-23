package pagevisit

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type Frontier interface {
	Owe(tx *vault.Txn, pageVisits []pendingpagevisit.PendingPageVisit) error
}
