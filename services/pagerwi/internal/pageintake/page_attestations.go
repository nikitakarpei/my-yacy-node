package pageintake

import (
	"context"
	"encoding/json"

	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type PageAttestations interface {
	AttestationOf(
		ctx context.Context,
		offer pagescrapecontract.PageOfferMessage,
		page pagescrapecontract.OfferedPage,
		pageRWI pagerwicontract.PageRWI,
	) (json.RawMessage, bool)
}
