package crawls_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
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

const retention = 7 * 24 * time.Hour

var (
	startedAt         = time.UnixMilli(1_700_000_000_000)
	errEngineRefusing = errors.New("engine refuses")
	errPortRefusing   = errors.New("port refuses")
)

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

type refusingRecords struct {
	vault.EngineRecords
}

func (refusingRecords) Get([]byte) ([]byte, error) {
	return nil, errEngineRefusing
}

func (refusingRecords) Scan(vault.KeyRange, func(key, value []byte) (bool, error)) error {
	return errEngineRefusing
}

type clock struct {
	guard  sync.Mutex
	now    time.Time
	sleeps chan struct{}
	wakes  chan struct{}
}

func (c *clock) Now() time.Time {
	c.guard.Lock()
	defer c.guard.Unlock()
	return c.now
}

func (c *clock) Sleep(ctx context.Context, _ time.Duration) error {
	select {
	case c.sleeps <- struct{}{}:
	case <-ctx.Done():
		return fmt.Errorf("context: %w", ctx.Err())
	}
	select {
	case <-c.wakes:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context: %w", ctx.Err())
	}
}

func (c *clock) advanceBy(d time.Duration) {
	c.guard.Lock()
	defer c.guard.Unlock()
	c.now = c.now.Add(d)
}

func (c *clock) runRemovalPass() {
	c.wakes <- struct{}{}
	<-c.sleeps
}

type reportedChange struct {
	change  string
	crawlID crawlid.CrawlID
}

type reportedChanges struct {
	reported []reportedChange
	refusing bool
}

func (r *reportedChanges) report(change string, crawlID crawlid.CrawlID) error {
	if r.refusing {
		return errPortRefusing
	}
	r.reported = append(r.reported, reportedChange{change: change, crawlID: crawlID})
	return nil
}

func (r *reportedChanges) CrawlFinished(_ *vault.Txn, crawlID crawlid.CrawlID) error {
	return r.report("finished", crawlID)
}

func (r *reportedChanges) CrawlForgotten(_ *vault.Txn, crawlID crawlid.CrawlID) error {
	return r.report("forgotten", crawlID)
}

func (r *reportedChanges) take() []reportedChange {
	taken := r.reported
	r.reported = nil
	return taken
}

type scene struct {
	storage *vault.Vault
	crawls  *crawls.Crawls
	engine  *refusingEngine
	clock   *clock
	changes *reportedChanges
}

func keptCrawls(t *testing.T) scene {
	t.Helper()
	engine := &refusingEngine{Engine: memoryvault.OpenEngine(0)}
	sceneClock := &clock{now: startedAt, sleeps: make(chan struct{}), wakes: make(chan struct{})}
	storage, err := vault.New(engine, nil, nil, sceneClock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	changes := &reportedChanges{}
	kept, err := crawls.Open(t.Context(), storage, retention, changes, changes, sceneClock)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	<-sceneClock.sleeps
	return scene{
		storage: storage,
		crawls:  kept,
		engine:  engine,
		clock:   sceneClock,
		changes: changes,
	}
}

func orderWithID(t *testing.T, orderID string) acceptedorder.AcceptedOrder {
	t.Helper()
	accepted, err := acceptedorder.AcceptedOrderFrom(yacycrawlcontract.CrawlOrder{
		OrderID: orderID,
		Profile: yacycrawlcontract.CrawlProfile{
			Scope:    yacycrawlcontract.ScopeDomain,
			MaxDepth: 2,
		},
		SeedURLs: []canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")},
	})
	if err != nil {
		t.Fatalf("accept order: %v", err)
	}
	return accepted
}

func (s scene) update(t *testing.T, fn func(tx *vault.Txn) error) {
	t.Helper()
	if err := s.storage.Update(t.Context(), fn); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (s scene) view(t *testing.T, fn func(tx *vault.Txn) error) {
	t.Helper()
	if err := s.storage.View(t.Context(), fn); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func (s scene) startOrFind(t *testing.T, orderID string) (crawlid.CrawlID, bool) {
	t.Helper()
	var crawlID crawlid.CrawlID
	var started bool
	s.update(t, func(tx *vault.Txn) error {
		var err error
		crawlID, started, err = s.crawls.Start(tx, orderWithID(t, orderID))
		return err
	})
	return crawlID, started
}

func (s scene) start(t *testing.T, orderID string) crawlid.CrawlID {
	t.Helper()
	crawlID, started := s.startOrFind(t, orderID)
	if !started {
		t.Fatalf("order %s started no crawl", orderID)
	}
	return crawlID
}

func (s scene) settle(t *testing.T, crawlID crawlid.CrawlID) {
	t.Helper()
	s.update(t, func(tx *vault.Txn) error { return s.crawls.Settle(tx, crawlID) })
}

func (s scene) crawlOf(t *testing.T, crawlID crawlid.CrawlID) (crawls.Crawl, bool) {
	t.Helper()
	var crawl crawls.Crawl
	var found bool
	s.view(t, func(tx *vault.Txn) error {
		var err error
		crawl, found, err = s.crawls.CrawlOf(tx, crawlID)
		return err
	})
	return crawl, found
}

func TestAStartedCrawlRunsWithItsOrderAndStart(t *testing.T) {
	s := keptCrawls(t)

	crawl, found := s.crawlOf(t, s.start(t, "o1"))

	if !found || crawl.Order.OrderID != "o1" ||
		crawl.State != yacycrawlcontract.CrawlStateRunning ||
		!crawl.StartedAt.Equal(startedAt) ||
		!crawl.FinishedAt.IsZero() {
		t.Fatalf("crawl = %+v (found %v), want the running crawl of o1", crawl, found)
	}
}

func TestAnOrderSentAgainFindsItsCrawlWhileItIsKept(t *testing.T) {
	s := keptCrawls(t)
	first := s.start(t, "o1")

	again, startedAgain := s.startOrFind(t, "o1")
	s.settle(t, first)
	afterFinish, startedAfterFinish := s.startOrFind(t, "o1")

	if startedAgain || again != first || startedAfterFinish || afterFinish != first {
		t.Fatalf("Start again = %s (%v), after finish %s (%v); want crawl %s found twice",
			again, startedAgain, afterFinish, startedAfterFinish, first)
	}
	if s.start(t, "o2") == first {
		t.Fatal("two orders share one crawl")
	}
}

func TestTheOrderOfACrawlReadsBack(t *testing.T) {
	s := keptCrawls(t)
	crawlID := s.start(t, "o1")
	var order acceptedorder.AcceptedOrder

	s.view(t, func(tx *vault.Txn) error {
		var err error
		order, _, err = s.crawls.OrderOf(tx, crawlID)
		return err
	})

	if order.OrderID() != "o1" {
		t.Fatalf("order %s, want o1", order.OrderID())
	}
}

func TestAnUnknownCrawlIsNotFoundAndChangesNothing(t *testing.T) {
	s := keptCrawls(t)
	unknown := crawlid.CrawlID{9}
	var orderFound bool

	s.update(t, func(tx *vault.Txn) error {
		var err error
		if _, orderFound, err = s.crawls.OrderOf(tx, unknown); err != nil {
			return err
		}
		return s.crawls.Settle(tx, unknown)
	})

	if orderFound || len(s.changes.take()) != 0 {
		t.Fatal("a crawl never started was found")
	}
}

func TestASettledCrawlSucceedsOnceAndReportsItsFinish(t *testing.T) {
	s := keptCrawls(t)
	crawlID := s.start(t, "o1")
	s.clock.advanceBy(time.Hour)

	s.settle(t, crawlID)
	s.settle(t, crawlID)

	succeeded, _ := s.crawlOf(t, crawlID)
	if succeeded.State != yacycrawlcontract.CrawlStateSucceeded ||
		!succeeded.FinishedAt.Equal(startedAt.Add(time.Hour)) {
		t.Fatalf("settled %+v, want succeeded an hour after start", succeeded)
	}
	if changes := s.changes.take(); !slices.Equal(
		changes,
		[]reportedChange{{change: "finished", crawlID: crawlID}},
	) {
		t.Fatalf("changes = %v, want one finish", changes)
	}
}

func TestAFinishedCrawlIsForgottenAfterTheRetentionAndItsOrderStartsAgain(t *testing.T) {
	s := keptCrawls(t)
	crawlID := s.start(t, "o1")
	s.clock.advanceBy(retention)
	s.settle(t, crawlID)
	s.changes.take()

	s.clock.advanceBy(retention - time.Second)
	s.clock.runRemovalPass()
	_, keptBeforeRetention := s.crawlOf(t, crawlID)
	s.clock.advanceBy(time.Second)
	again, startedAgain := s.startOrFind(t, "o1")
	s.clock.runRemovalPass()

	if !keptBeforeRetention || !startedAgain || again == crawlID {
		t.Fatalf("kept before retention %v, started again %v as %s",
			keptBeforeRetention, startedAgain, again)
	}
	if changes := s.changes.take(); !slices.Equal(
		changes,
		[]reportedChange{{change: "forgotten", crawlID: crawlID}},
	) {
		t.Fatalf("changes = %v, want the crawl forgotten", changes)
	}
}

func TestAFinishThePortRefusesFails(t *testing.T) {
	s := keptCrawls(t)
	crawlID := s.start(t, "o1")
	s.changes.refusing = true

	err := s.storage.Update(t.Context(), func(tx *vault.Txn) error {
		return s.crawls.Settle(tx, crawlID)
	})

	if !errors.Is(err, errPortRefusing) {
		t.Fatalf("Settle = %v, want the port refusal", err)
	}
	if crawl, _ := s.crawlOf(t, crawlID); crawl.State != yacycrawlcontract.CrawlStateRunning {
		t.Fatalf("crawl = %+v, want it still running", crawl)
	}
}

func TestCrawlReadsTheStorageRefusesFail(t *testing.T) {
	s := keptCrawls(t)
	crawlID := s.start(t, "o1")
	s.engine.refusing.Store(true)

	startErr := s.storage.Update(t.Context(), func(tx *vault.Txn) error {
		_, _, err := s.crawls.Start(tx, orderWithID(t, "o2"))
		return err
	})
	readErr := s.storage.View(t.Context(), func(tx *vault.Txn) error {
		_, _, err := s.crawls.CrawlOf(tx, crawlID)
		return err
	})
	settleErr := s.storage.Update(t.Context(), func(tx *vault.Txn) error {
		return s.crawls.Settle(tx, crawlID)
	})

	for name, err := range map[string]error{
		"start": startErr, "read": readErr, "settle": settleErr,
	} {
		if !errors.Is(err, errEngineRefusing) {
			t.Errorf("%s = %v, want the refusal", name, err)
		}
	}
}
