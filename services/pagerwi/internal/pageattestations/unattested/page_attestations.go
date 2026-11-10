// Package unattested publishes each RWI without a page attestation.
package unattested

import (
	"context"
	"encoding/json"

	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type PageAttestations struct{}

func (PageAttestations) AttestationOf(
	context.Context,
	pagescrapecontract.PageOfferMessage,
	pagescrapecontract.OfferedPage,
	pagerwicontract.PageRWI,
) (json.RawMessage, bool) {
	return nil, true
}
