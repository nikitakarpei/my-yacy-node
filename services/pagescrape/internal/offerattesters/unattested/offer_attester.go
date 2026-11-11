// Package unattested dates each fetched page by its clock and attests no page offer.
package unattested

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type Clock interface {
	Now() time.Time
}

type OfferAttester struct {
	clock Clock
}

func New(clock Clock) OfferAttester {
	return OfferAttester{clock: clock}
}

func (a OfferAttester) FetchTimeOf(
	_ context.Context,
	_ redirectfollowingfetch.LandedFetch,
) time.Time {
	return a.clock.Now()
}

func (OfferAttester) Attest(
	_ context.Context,
	_ pagescrapecontract.PageOfferMessage,
	_ redirectfollowingfetch.LandedFetch,
) (attestedstatements.Bundle, bool) {
	return nil, true
}
