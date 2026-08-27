package jetstream_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/acceptedorders/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
)

func acceptedOrders(t *testing.T) *jetstream.Orders {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	ctx := context.Background()
	if err := jetstream.Ensure(ctx, js, jetstream.BucketSpec{}); err != nil {
		t.Fatalf("ensure bucket: %v", err)
	}
	bucket, err := js.KeyValue(ctx, jetstream.BucketName)
	if err != nil {
		t.Fatalf("open bucket: %v", err)
	}
	return jetstream.New(bucket)
}

func order(t *testing.T) yacycrawlcontract.CrawlOrder {
	t.Helper()
	return yacycrawlcontract.CrawlOrder{
		OrderID: "o1",
		Profile: yacycrawlcontract.CrawlProfile{
			Scope:           yacycrawlcontract.ScopeDomain,
			URLMustMatch:    yacycrawlcontract.MatchAll,
			MaxDepth:        2,
			MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
		},
		SeedURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://host/"),
		},
	}
}

func acceptedOrderOf(t *testing.T, sent yacycrawlcontract.CrawlOrder) acceptedorder.AcceptedOrder {
	t.Helper()
	accepted, err := acceptedorder.AcceptedOrderFrom(sent)
	if err != nil {
		t.Fatalf("accept order: %v", err)
	}
	return accepted
}

func TestOrderOfReadsBackTheAcceptedOrder(t *testing.T) {
	orders := acceptedOrders(t)
	ctx := context.Background()
	accepted := acceptedOrderOf(t, order(t))

	if err := orders.Accept(ctx, accepted); err != nil {
		t.Fatalf("accept: %v", err)
	}
	read, err := orders.OrderOf(ctx, accepted.OrderID())
	if err != nil {
		t.Fatalf("order of: %v", err)
	}

	if read.CrawlOrder().OrderID != accepted.CrawlOrder().OrderID ||
		read.CrawlOrder().Profile != accepted.CrawlOrder().Profile {
		t.Fatalf("read %+v, want %+v", read.CrawlOrder(), accepted.CrawlOrder())
	}
}

func TestAcceptTheSameOrderTwiceKeepsIt(t *testing.T) {
	orders := acceptedOrders(t)
	ctx := context.Background()
	accepted := acceptedOrderOf(t, order(t))

	if err := orders.Accept(ctx, accepted); err != nil {
		t.Fatalf("first accept: %v", err)
	}
	if err := orders.Accept(ctx, accepted); err != nil {
		t.Fatalf("second accept: %v", err)
	}
	if _, err := orders.OrderOf(ctx, accepted.OrderID()); err != nil {
		t.Fatalf("order of: %v", err)
	}
}

func TestOrderOfAnUnknownOrderFails(t *testing.T) {
	if _, err := acceptedOrders(t).OrderOf(context.Background(), "missing"); err == nil {
		t.Fatal("an order the crawler never accepted should not read back")
	}
}
