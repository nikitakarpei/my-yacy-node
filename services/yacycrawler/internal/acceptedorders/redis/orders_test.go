package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/acceptedorders/redis"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
)

const retention = time.Hour

func acceptedOrdersOn(server *miniredis.Miniredis) *redis.Orders {
	return redis.New(goredis.NewClient(&goredis.Options{Addr: server.Addr()}), retention)
}

func acceptedOrders(t *testing.T) *redis.Orders {
	t.Helper()
	return acceptedOrdersOn(miniredis.RunT(t))
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

	if err := orders.Keep(ctx, accepted); err != nil {
		t.Fatalf("keep: %v", err)
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

func TestKeepingTheSameOrderTwiceKeepsIt(t *testing.T) {
	orders := acceptedOrders(t)
	ctx := context.Background()
	accepted := acceptedOrderOf(t, order(t))

	if err := orders.Keep(ctx, accepted); err != nil {
		t.Fatalf("first keep: %v", err)
	}
	if err := orders.Keep(ctx, accepted); err != nil {
		t.Fatalf("second keep: %v", err)
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

func TestAnOrderIsForgottenOnceItsRetentionPasses(t *testing.T) {
	server := miniredis.RunT(t)
	orders := acceptedOrdersOn(server)
	ctx := context.Background()
	accepted := acceptedOrderOf(t, order(t))
	if err := orders.Keep(ctx, accepted); err != nil {
		t.Fatalf("keep: %v", err)
	}

	server.FastForward(retention)

	if _, err := orders.OrderOf(ctx, accepted.OrderID()); err == nil {
		t.Fatal("an order past its retention should not read back")
	}
}
