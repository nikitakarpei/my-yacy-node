package visitintake

import (
	"github.com/google/uuid"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func crawlOrderFor(
	seedURL yacycrawlcontract.CanonicalURL,
	profile yacycrawlcontract.CrawlProfile,
) yacycrawlcontract.CrawlOrder {
	return yacycrawlcontract.CrawlOrder{
		OrderID:  uuid.NewString(),
		Profile:  profile,
		SeedURLs: []yacycrawlcontract.CanonicalURL{seedURL},
	}
}
