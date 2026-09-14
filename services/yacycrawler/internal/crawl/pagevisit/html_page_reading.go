package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
)

type HTMLPageReading interface {
	ReadingOfPage(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		page pagefetch.FetchedPage,
	) (pagehtmlreading.Reading, error)
}
