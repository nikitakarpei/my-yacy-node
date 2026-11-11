package pageintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type RWIAttester interface {
	Attest(
		ctx context.Context,
		offer pagescrapecontract.PageOfferMessage,
		page pagescrapecontract.OfferedPage,
		pageRWI pagerwicontract.PageRWI,
	) (attestedstatements.Bundle, bool)
}
