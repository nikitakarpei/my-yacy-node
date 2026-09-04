package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerefusals"
)

type HTMLPageReading interface {
	ReadingOfPage(
		ctx context.Context,
		page pagefetch.FetchedPage,
		ignored pagerefusals.IgnoredRefusals,
	) (pagehtmlreading.Reading, error)
}
