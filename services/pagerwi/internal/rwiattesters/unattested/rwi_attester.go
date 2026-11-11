// Package unattested publishes each RWI without a page attestation.
package unattested

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type RWIAttester struct{}

func (RWIAttester) Attest(
	context.Context,
	pagescrapecontract.PageOfferMessage,
	pagescrapecontract.OfferedPage,
	pagerwicontract.PageRWI,
) (attestedstatements.Bundle, bool) {
	return nil, true
}
