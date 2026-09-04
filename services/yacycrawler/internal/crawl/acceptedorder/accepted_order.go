// Package acceptedorder holds a crawl order in the shape the crawler works
// from: the order as the operator sent it, and the rule that decides which
// URLs it admits.
package acceptedorder

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/profileadmission"
)

type AcceptedOrder struct {
	order     yacycrawlcontract.CrawlOrder
	admission *profileadmission.Admission
}

func AcceptedOrderFrom(order yacycrawlcontract.CrawlOrder) (AcceptedOrder, error) {
	admission, err := profileadmission.New(order.Profile, order.SeedURLs)
	if err != nil {
		return AcceptedOrder{}, fmt.Errorf("read the profile of order %s: %w", order.OrderID, err)
	}
	return AcceptedOrder{order: order, admission: admission}, nil
}

func (o AcceptedOrder) OrderID() string {
	return o.order.OrderID
}

func (o AcceptedOrder) SeedURLs() []canonicalurl.CanonicalURL {
	return o.order.SeedURLs
}

func (o AcceptedOrder) MaxPagesPerHost() int {
	return o.order.Profile.MaxPagesPerHost
}

func (o AcceptedOrder) Admits(url canonicalurl.CanonicalURL, depth int) bool {
	return o.admission.Admits(url, depth)
}

func (o AcceptedOrder) CrawlOrder() yacycrawlcontract.CrawlOrder {
	return o.order
}
