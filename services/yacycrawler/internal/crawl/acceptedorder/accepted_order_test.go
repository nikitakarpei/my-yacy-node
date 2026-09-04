package acceptedorder_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
)

func crawlOrder(t *testing.T) yacycrawlcontract.CrawlOrder {
	t.Helper()
	return yacycrawlcontract.CrawlOrder{
		OrderID: "o1",
		Profile: yacycrawlcontract.CrawlProfile{
			Scope:           yacycrawlcontract.ScopeDomain,
			URLMustMatch:    yacycrawlcontract.MatchAll,
			MaxDepth:        2,
			MaxPagesPerHost: 7,
		},
		SeedURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://host/one"),
		},
	}
}

func acceptedOrder(t *testing.T, order yacycrawlcontract.CrawlOrder) acceptedorder.AcceptedOrder {
	t.Helper()
	accepted, err := acceptedorder.AcceptedOrderFrom(order)
	if err != nil {
		t.Fatalf("accept order: %v", err)
	}
	return accepted
}

func TestAnAcceptedOrderCarriesWhatTheOperatorSent(t *testing.T) {
	order := crawlOrder(t)

	accepted := acceptedOrder(t, order)

	if accepted.OrderID() != "o1" {
		t.Errorf("order = %q, want the order the operator sent", accepted.OrderID())
	}
	if len(accepted.SeedURLs()) != 1 {
		t.Errorf("seeds = %v, want the seeds the operator sent", accepted.SeedURLs())
	}
	if accepted.MaxPagesPerHost() != 7 {
		t.Errorf("max pages per host = %d, want 7", accepted.MaxPagesPerHost())
	}
	if accepted.CrawlOrder().Profile.MaxDepth != 2 {
		t.Errorf("crawl order = %+v, want the order the operator sent", accepted.CrawlOrder())
	}
}

func TestAnAcceptedOrderAdmitsWhatItsProfileAdmits(t *testing.T) {
	accepted := acceptedOrder(t, crawlOrder(t))

	if !accepted.Admits(canonicalurltest.CanonicalURLOf(t, "http://host/two"), 1) {
		t.Error("a url on a seed host within the depth is admitted")
	}
	if accepted.Admits(canonicalurltest.CanonicalURLOf(t, "http://elsewhere/two"), 1) {
		t.Error("a url off every seed host is not admitted under domain scope")
	}
	if accepted.Admits(canonicalurltest.CanonicalURLOf(t, "http://host/deep"), 3) {
		t.Error("a url beyond the profile depth is not admitted")
	}
}

func TestAnOrderNamingAnUnreadablePatternIsNotAccepted(t *testing.T) {
	order := crawlOrder(t)
	order.Profile.URLMustNotMatch = "([unclosed"

	if _, err := acceptedorder.AcceptedOrderFrom(order); err == nil {
		t.Fatal("an order naming an unreadable pattern cannot be accepted")
	}
}
