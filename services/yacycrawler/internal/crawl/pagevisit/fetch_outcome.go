package pagevisit

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
)

type FetchOutcome struct {
	Status        FetchStatus
	DeferFor      time.Duration
	Page          fetchedpage.Page
	RedirectChain []string
	Version       PageVersion
}
