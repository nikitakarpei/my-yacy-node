package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
)

type HTMLPageReading interface {
	ReadingOfPage(
		ctx context.Context,
		page pagefetch.FetchedPage,
	) (pagehtmlreading.Reading, error)
}
