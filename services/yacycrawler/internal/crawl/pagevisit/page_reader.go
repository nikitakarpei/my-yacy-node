package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagereading"
)

type PageReader interface {
	ReadingOfPage(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		page pagefetch.FetchedPage,
	) (pagereading.Reading, error)
}
