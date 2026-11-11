package scrapeintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type OfferAttester interface {
	FetchTimeOf(ctx context.Context, landed redirectfollowingfetch.LandedFetch) time.Time
	Attest(
		ctx context.Context,
		offer pagescrapecontract.PageOfferMessage,
		landed redirectfollowingfetch.LandedFetch,
	) (attestedstatements.Bundle, bool)
}
