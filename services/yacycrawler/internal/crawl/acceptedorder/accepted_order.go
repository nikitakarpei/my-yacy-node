// Package acceptedorder holds a crawl order in the shape the crawler works
// from: the order as the operator sent it, and the rule that decides which
// URLs it admits.
package acceptedorder

import (
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/profileadmission"
)

var (
	errNoSeedURLs           = errors.New("the order has no seed URLs")
	errNegativePagesPerHost = errors.New("MaxPagesPerHost is below 0")
	errNegativeRecrawlGrace = errors.New("RecrawlGrace is below 0")
	errRecrawlGracePastMax  = fmt.Errorf(
		"RecrawlGrace is past %s", yacycrawlcontract.MaxRecrawlGrace,
	)
)

type AcceptedOrder struct {
	order     yacycrawlcontract.CrawlOrder
	admission *profileadmission.Admission
}

func AcceptedOrderFrom(order yacycrawlcontract.CrawlOrder) (AcceptedOrder, error) {
	if len(order.SeedURLs) == 0 {
		return AcceptedOrder{}, fmt.Errorf("read order %s: %w", order.OrderID, errNoSeedURLs)
	}
	if order.Profile.MaxPagesPerHost < 0 {
		return AcceptedOrder{}, fmt.Errorf(
			"read the profile of order %s: %w",
			order.OrderID,
			errNegativePagesPerHost,
		)
	}
	if err := readableRecrawlGrace(order.Profile.RecrawlGrace); err != nil {
		return AcceptedOrder{}, fmt.Errorf("read the profile of order %s: %w", order.OrderID, err)
	}
	admission, err := profileadmission.New(order.Profile, order.SeedURLs)
	if err != nil {
		return AcceptedOrder{}, fmt.Errorf("read the profile of order %s: %w", order.OrderID, err)
	}
	return AcceptedOrder{order: order, admission: admission}, nil
}

func readableRecrawlGrace(recrawlGrace time.Duration) error {
	switch {
	case recrawlGrace < yacycrawlcontract.NoRecrawlGrace:
		return errNegativeRecrawlGrace
	case recrawlGrace > yacycrawlcontract.MaxRecrawlGrace:
		return errRecrawlGracePastMax
	default:
		return nil
	}
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

func (o AcceptedOrder) RecrawlGrace() time.Duration {
	return o.order.Profile.RecrawlGrace
}

func (o AcceptedOrder) Admits(url canonicalurl.CanonicalURL, depth int) bool {
	return o.admission.Admits(url, depth)
}

func (o AcceptedOrder) CrawlOrder() yacycrawlcontract.CrawlOrder {
	return o.order
}
