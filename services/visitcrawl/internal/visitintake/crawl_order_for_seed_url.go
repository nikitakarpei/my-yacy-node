package visitintake

import (
	"github.com/google/uuid"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func crawlOrderFor(
	seedURL canonicalurl.CanonicalURL,
	profile yacycrawlcontract.CrawlProfile,
) yacycrawlcontract.CrawlOrder {
	return yacycrawlcontract.CrawlOrder{
		OrderID:  uuid.NewString(),
		Profile:  profile,
		SeedURLs: []canonicalurl.CanonicalURL{seedURL},
	}
}
