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

type OfferedPages struct {
	clock Clock
}

func New(clock Clock) OfferedPages {
	return OfferedPages{clock: clock}
}

func (p OfferedPages) OfferedPageFrom(
	_ context.Context,
	request pagescrapecontract.ScrapeRequest,
	landed redirectfollowingfetch.LandedFetch,
) (pagescrapecontract.OfferedPage, bool) {
	return pagescrapecontract.OfferedPageFrom(
		request, landed.Outcome.Page, landed.Outcome.Version, landed.URL, p.clock.Now(),
	), true
}
