package visitintake

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func crawlOrderFromVisit(
	visitedPage string,
	profile yacycrawlcontract.CrawlProfile,
) (yacycrawlcontract.CrawlOrder, error) {
	canonicalURL, err := yacycrawlcontract.CanonicalURLOf(visitedPage)
	if err != nil {
		return yacycrawlcontract.CrawlOrder{}, fmt.Errorf("canonicalize visited page: %w", err)
	}
	return yacycrawlcontract.CrawlOrder{
		OrderID:  uuid.NewString(),
		Profile:  profile,
		SeedURLs: []yacycrawlcontract.CanonicalURL{canonicalURL},
	}, nil
}
