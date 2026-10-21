package vault_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	acceptedordersvault "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/acceptedorders/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/retainedrecords"
)

const keptFor = time.Hour

var (
	acceptedAt        = time.UnixMilli(1_700_000_000_000)
	errEngineRefusing = errors.New("engine refuses")
)

type settableClock struct {
	now time.Time
}

func (clock *settableClock) Now() time.Time { return clock.now }

func (*settableClock) Sleep(context.Context, time.Duration) error { return context.Canceled }

type silentExpiryObserver struct{}

func (silentExpiryObserver) ExpiredRecordsForgotten(context.Context, string, int) {}

func (silentExpiryObserver) ExpiredRecordsNotForgotten(context.Context, string, error) {}

type refusingEngine struct {
	vault.Engine
	refusing bool
}

func (engine *refusingEngine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if engine.refusing {
		return errEngineRefusing
	}
	return engine.Engine.Update(ctx, fn)
}

func (engine *refusingEngine) View(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if engine.refusing {
		return errEngineRefusing
	}
	return engine.Engine.View(ctx, fn)
}

type scene struct {
	orders *acceptedordersvault.Orders
	clock  *settableClock
	engine *refusingEngine
}

func acceptedOrders(t *testing.T) scene {
	t.Helper()
	engine := &refusingEngine{Engine: memoryvault.OpenEngine(0)}
	storage, err := vault.New(engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	clock := &settableClock{now: acceptedAt}
	orders, err := acceptedordersvault.New(
		t.Context(),
		storage,
		retainedrecords.New(storage, clock, silentExpiryObserver{}),
		clock,
		keptFor,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return scene{orders: orders, clock: clock, engine: engine}
}

func acceptedOrderWithID(t *testing.T, orderID string) acceptedorder.AcceptedOrder {
	t.Helper()
	accepted, err := acceptedorder.AcceptedOrderFrom(yacycrawlcontract.CrawlOrder{
		OrderID: orderID,
		Profile: yacycrawlcontract.CrawlProfile{
			Scope:           yacycrawlcontract.ScopeDomain,
			URLMustMatch:    yacycrawlcontract.MatchAll,
			MaxDepth:        2,
			MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
		},
		SeedURLs: []canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")},
	})
	if err != nil {
		t.Fatalf("accept order: %v", err)
	}
	return accepted
}

func (s scene) keep(t *testing.T, order acceptedorder.AcceptedOrder) crawlid.CrawlID {
	t.Helper()
	crawlID, err := s.orders.Keep(t.Context(), order)
	if err != nil {
		t.Fatalf("Keep: %v", err)
	}
	return crawlID
}

func (s scene) orderOf(t *testing.T, crawlID crawlid.CrawlID) (acceptedorder.AcceptedOrder, bool) {
	t.Helper()
	order, found, err := s.orders.OrderOf(t.Context(), crawlID)
	if err != nil {
		t.Fatalf("OrderOf: %v", err)
	}
	return order, found
}

func TestOrderOfReadsBackTheOrderOfItsCrawl(t *testing.T) {
	s := acceptedOrders(t)
	accepted := acceptedOrderWithID(t, "o1")

	read, found := s.orderOf(t, s.keep(t, accepted))

	if !found || read.OrderID() != "o1" ||
		read.CrawlOrder().Profile != accepted.CrawlOrder().Profile {
		t.Fatalf("read %+v (found %v), want %+v", read.CrawlOrder(), found, accepted.CrawlOrder())
	}
}

func TestAnOrderKeptTwiceKeepsItsCrawl(t *testing.T) {
	s := acceptedOrders(t)
	accepted := acceptedOrderWithID(t, "o1")

	first := s.keep(t, accepted)
	if again := s.keep(t, accepted); again != first {
		t.Fatalf("second keep began crawl %s, want crawl %s again", again, first)
	}
}

func TestTwoOrdersBeginTwoCrawls(t *testing.T) {
	s := acceptedOrders(t)

	if s.keep(t, acceptedOrderWithID(t, "o1")) == s.keep(t, acceptedOrderWithID(t, "o2")) {
		t.Fatal("two orders share one crawl")
	}
}

func TestTheOrderOfAnUnknownCrawlIsNotFound(t *testing.T) {
	if _, found := acceptedOrders(t).orderOf(t, crawlid.CrawlID{2}); found {
		t.Fatal("a crawl the crawler never began has an order")
	}
}

func TestAnOrderIsForgottenOnceItsRetentionPasses(t *testing.T) {
	s := acceptedOrders(t)
	accepted := acceptedOrderWithID(t, "o1")
	crawlID := s.keep(t, accepted)

	s.clock.now = acceptedAt.Add(keptFor)

	if _, found := s.orderOf(t, crawlID); found {
		t.Fatal("an order past its retention reads back")
	}
	if s.keep(t, accepted) == crawlID {
		t.Fatal("an order sent after its retention resumes the forgotten crawl")
	}
}

func TestAKeepTheStorageRefusesFails(t *testing.T) {
	s := acceptedOrders(t)
	s.engine.refusing = true

	if _, err := s.orders.Keep(t.Context(), acceptedOrderWithID(t, "o1")); !errors.Is(
		err, errEngineRefusing,
	) {
		t.Fatalf("Keep err = %v, want the refusal", err)
	}
}

func TestAReadTheStorageRefusesFails(t *testing.T) {
	s := acceptedOrders(t)
	crawlID := s.keep(t, acceptedOrderWithID(t, "o1"))
	s.engine.refusing = true

	if _, _, err := s.orders.OrderOf(t.Context(), crawlID); !errors.Is(err, errEngineRefusing) {
		t.Fatalf("OrderOf err = %v, want the refusal", err)
	}
}
