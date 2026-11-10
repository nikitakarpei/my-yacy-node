// Package unattested offers each fetched page without an attestation, as fetched at the time of
// its clock.
package unattested

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type Clock interface {
	Now() time.Time
}

type OfferIssuer struct {
	clock Clock
}

func New(clock Clock) OfferIssuer {
	return OfferIssuer{clock: clock}
}

func (p OfferIssuer) Issue(
	_ context.Context,
	request pagescrapecontract.ScrapeRequest,
	landed redirectfollowingfetch.LandedFetch,
) (pagescrapecontract.PageOfferMessage, bool) {
	return pagescrapecontract.PageOfferMessageOf(pagescrapecontract.OfferedPageFrom(
		request, landed.Outcome.Page, landed.Outcome.Version, landed.URL, p.clock.Now(),
	)), true
}
