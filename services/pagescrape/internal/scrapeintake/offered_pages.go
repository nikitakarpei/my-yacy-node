package scrapeintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type OfferedPages interface {
	OfferedPageFrom(
		ctx context.Context,
		request pagescrapecontract.ScrapeRequest,
		landed redirectfollowingfetch.LandedFetch,
	) (pagescrapecontract.OfferedPage, bool)
}
