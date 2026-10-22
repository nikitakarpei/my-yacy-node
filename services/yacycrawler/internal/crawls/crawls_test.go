package crawls_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawls"
)

const keptFor = time.Hour

var (
	acceptedAt        = time.UnixMilli(1_700_000_000_000)
	errEngineRefusing = errors.New("engine refuses")
)

type settableClock struct {
	now atomic.Pointer[time.Time]
}

func newSettableClock(now time.Time) *settableClock {
	clock := &settableClock{}
	clock.set(now)
	return clock
}

func (clock *settableClock) Now() time.Time { return *clock.now.Load() }

func (clock *settableClock) set(now time.Time) { clock.now.Store(&now) }

func (*settableClock) Sleep(context.Context, time.Duration) error { return context.Canceled }

type refusingEngine struct {
	vault.Engine
	refusing atomic.Bool
}

func (engine *refusingEngine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	return engine.Engine.Update(ctx, func(etx vault.EngineTxn) error {
		return fn(engine.txnOf(etx))
	})
}

func (engine *refusingEngine) View(ctx context.Context, fn func(vault.EngineTxn) error) error {
	return engine.Engine.View(ctx, func(etx vault.EngineTxn) error {
		return fn(engine.txnOf(etx))
	})
}

func (engine *refusingEngine) txnOf(etx vault.EngineTxn) vault.EngineTxn {
	if engine.refusing.Load() {
		return refusingTxn{EngineTxn: etx}
	}
	return etx
}

type refusingTxn struct {
	vault.EngineTxn
}

func (txn refusingTxn) Records(name vault.Name) vault.EngineRecords {
	return refusingRecords{EngineRecords: txn.EngineTxn.Records(name)}
}

func (txn refusingTxn) ExpiringRecords(name vault.Name) vault.EngineExpiringRecords {
	return refusingExpiringRecords{EngineExpiringRecords: txn.EngineTxn.ExpiringRecords(name)}
}

type refusingRecords struct {
	vault.EngineRecords
}

func (refusingRecords) Scan(vault.KeyRange, func(key, value []byte) (bool, error)) error {
	return errEngineRefusing
}

type refusingExpiringRecords struct {
	vault.EngineExpiringRecords
}

func (refusingExpiringRecords) Get([]byte, time.Time) ([]byte, error) {
	return nil, errEngineRefusing
}

type scene struct {
	storage *vault.Vault
	crawls  *crawls.Crawls
	clock   *settableClock
	engine  *refusingEngine
}

func acceptedOrders(t *testing.T) scene {
	t.Helper()
	engine := &refusingEngine{Engine: memoryvault.OpenEngine(0)}
	clock := newSettableClock(acceptedAt)
	storage, err := vault.New(engine, nil, nil, clock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	kept, err := crawls.Open(t.Context(), storage, keptFor)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return scene{storage: storage, crawls: kept, clock: clock, engine: engine}
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
	crawlID, err := s.keepWithin(t, order)
	if err != nil {
		t.Fatalf("Keep: %v", err)
	}
	return crawlID
}

func (s scene) keepWithin(
	t *testing.T,
	order acceptedorder.AcceptedOrder,
) (crawlid.CrawlID, error) {
	t.Helper()
	var crawlID crawlid.CrawlID
	err := s.storage.Update(t.Context(), func(tx *vault.Txn) error {
		var err error
		crawlID, err = s.crawls.Keep(tx, order)
		return err
	})
	return crawlID, err
}

func (s scene) orderOf(t *testing.T, crawlID crawlid.CrawlID) (acceptedorder.AcceptedOrder, bool) {
	t.Helper()
	order, found, err := s.orderWithin(t, crawlID)
	if err != nil {
		t.Fatalf("OrderOf: %v", err)
	}
	return order, found
}

func (s scene) orderWithin(
	t *testing.T,
	crawlID crawlid.CrawlID,
) (acceptedorder.AcceptedOrder, bool, error) {
	t.Helper()
	var order acceptedorder.AcceptedOrder
	var found bool
	err := s.storage.View(t.Context(), func(tx *vault.Txn) error {
		var err error
		order, found, err = s.crawls.OrderOf(tx, crawlID)
		return err
	})
	return order, found, err
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

	s.clock.set(acceptedAt.Add(keptFor))

	if _, found := s.orderOf(t, crawlID); found {
		t.Fatal("an order past its retention reads back")
	}
	if s.keep(t, accepted) == crawlID {
		t.Fatal("an order sent after its retention resumes the forgotten crawl")
	}
}

func TestAKeepTheStorageRefusesFails(t *testing.T) {
	s := acceptedOrders(t)
	s.engine.refusing.Store(true)

	if _, err := s.keepWithin(t, acceptedOrderWithID(t, "o1")); !errors.Is(
		err, errEngineRefusing,
	) {
		t.Fatalf("Keep err = %v, want the refusal", err)
	}
}

func TestAReadTheStorageRefusesFails(t *testing.T) {
	s := acceptedOrders(t)
	crawlID := s.keep(t, acceptedOrderWithID(t, "o1"))
	s.engine.refusing.Store(true)

	if _, _, err := s.orderWithin(t, crawlID); !errors.Is(err, errEngineRefusing) {
		t.Fatalf("OrderOf err = %v, want the refusal", err)
	}
}
