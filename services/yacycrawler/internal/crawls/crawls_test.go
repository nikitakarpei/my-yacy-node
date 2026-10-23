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

func (r *reportedChanges) CrawlPaused(_ *vault.Txn, crawlID crawlid.CrawlID) error {
	return r.report("paused", crawlID)
}

func (r *reportedChanges) CrawlResumed(_ *vault.Txn, crawlID crawlid.CrawlID) error {
	return r.report("resumed", crawlID)
}

func (r *reportedChanges) CrawlCancelling(_ *vault.Txn, crawlID crawlid.CrawlID) error {
	return r.report("cancelling", crawlID)
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
	var orderFound, paused, resumed, cancelled bool

	s.update(t, func(tx *vault.Txn) error {
		var err error
		if _, orderFound, err = s.crawls.OrderOf(tx, unknown); err != nil {
			return err
		}
		if _, paused, err = s.crawls.Pause(tx, unknown); err != nil {
			return err
		}
		if _, resumed, err = s.crawls.Resume(tx, unknown); err != nil {
			return err
		}
		if _, cancelled, err = s.crawls.Cancel(tx, unknown); err != nil {
			return err
		}
		return s.crawls.Settle(tx, unknown)
	})

	if orderFound || paused || resumed || cancelled || len(s.changes.take()) != 0 {
		t.Fatal("a crawl never started was found")
	}
}

func TestACrawlMovesThroughItsStatesAndReportsEachChange(t *testing.T) {
	s := keptCrawls(t)
	crawlID := s.start(t, "o1")
	var states []yacycrawlcontract.CrawlState

	for _, change := range []func(*vault.Txn, crawlid.CrawlID) (
		yacycrawlcontract.CrawlState, bool, error,
	){
		s.crawls.Resume, s.crawls.Pause, s.crawls.Pause, s.crawls.Resume,
		s.crawls.Cancel, s.crawls.Pause, s.crawls.Resume, s.crawls.Cancel,
	} {
		s.update(t, func(tx *vault.Txn) error {
			state, _, err := change(tx, crawlID)
			states = append(states, state)
			return err
		})
	}

	want := []yacycrawlcontract.CrawlState{
		yacycrawlcontract.CrawlStateRunning, yacycrawlcontract.CrawlStatePaused,
		yacycrawlcontract.CrawlStatePaused, yacycrawlcontract.CrawlStateRunning,
		yacycrawlcontract.CrawlStateCancelling, yacycrawlcontract.CrawlStateCancelling,
		yacycrawlcontract.CrawlStateCancelling, yacycrawlcontract.CrawlStateCancelling,
	}
	if !slices.Equal(states, want) {
		t.Fatalf("states = %v, want %v", states, want)
	}
	wantChanges := []reportedChange{
		{change: "paused", crawlID: crawlID},
		{change: "resumed", crawlID: crawlID},
		{change: "cancelling", crawlID: crawlID},
	}
	if changes := s.changes.take(); !slices.Equal(changes, wantChanges) {
		t.Fatalf("changes = %v, want %v", changes, wantChanges)
	}
}

func TestASettledCrawlFinishesByTheStateItWasIn(t *testing.T) {
	s := keptCrawls(t)
	succeeding, cancelling := s.start(t, "o1"), s.start(t, "o2")
	s.update(t, func(tx *vault.Txn) error {
		_, _, err := s.crawls.Cancel(tx, cancelling)
		return err
	})
	s.changes.take()
	s.clock.advanceBy(time.Hour)

	s.settle(t, succeeding)
	s.settle(t, cancelling)
	s.settle(t, succeeding)

	succeeded, _ := s.crawlOf(t, succeeding)
	cancelled, _ := s.crawlOf(t, cancelling)
	if succeeded.State != yacycrawlcontract.CrawlStateSucceeded ||
		cancelled.State != yacycrawlcontract.CrawlStateCancelled ||
		!succeeded.FinishedAt.Equal(startedAt.Add(time.Hour)) {
		t.Fatalf("settled %+v and %+v, want succeeded and cancelled an hour after start",
			succeeded, cancelled)
	}
	wantChanges := []reportedChange{
		{change: "finished", crawlID: succeeding},
		{change: "finished", crawlID: cancelling},
	}
	if changes := s.changes.take(); !slices.Equal(changes, wantChanges) {
		t.Fatalf("changes = %v, want %v", changes, wantChanges)
	}
}

func TestAFinishedCrawlRefusesEveryStateChange(t *testing.T) {
	s := keptCrawls(t)
	crawlID := s.start(t, "o1")
	s.settle(t, crawlID)
	s.changes.take()
	var states []yacycrawlcontract.CrawlState

	for _, change := range []func(*vault.Txn, crawlid.CrawlID) (
		yacycrawlcontract.CrawlState, bool, error,
	){s.crawls.Pause, s.crawls.Resume, s.crawls.Cancel} {
		s.update(t, func(tx *vault.Txn) error {
			state, _, err := change(tx, crawlID)
			states = append(states, state)
			return err
		})
	}

	if !slices.Equal(states, []yacycrawlcontract.CrawlState{
		yacycrawlcontract.CrawlStateSucceeded,
		yacycrawlcontract.CrawlStateSucceeded,
		yacycrawlcontract.CrawlStateSucceeded,
	}) || len(s.changes.take()) != 0 {
		t.Fatalf("states = %v, want succeeded with no change", states)
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

func TestCrawlsListInStateNewestFirstAndBackOldestFirst(t *testing.T) {
	s := keptCrawls(t)
	first, second, third := s.start(t, "o1"), s.start(t, "o2"), s.start(t, "o3")
	paused, finished := s.start(t, "o4"), s.start(t, "o5")
	s.update(t, func(tx *vault.Txn) error {
		_, _, err := s.crawls.Pause(tx, paused)
		return err
	})
	s.settle(t, finished)
	var newest, older, newer, succeeded []crawlid.CrawlID
	var running int

	s.view(t, func(tx *vault.Txn) error {
		var err error
		state := yacycrawlcontract.CrawlStateRunning
		if newest, err = s.crawls.CrawlIDsOlderThan(tx, state, crawlid.CrawlID{}, 2); err != nil {
			return err
		}
		if older, err = s.crawls.CrawlIDsOlderThan(tx, state, second, 5); err != nil {
			return err
		}
		if newer, err = s.crawls.CrawlIDsNewerThan(tx, state, first, 5); err != nil {
			return err
		}
		if succeeded, err = s.crawls.CrawlIDsOlderThan(
			tx, yacycrawlcontract.CrawlStateSucceeded, crawlid.CrawlID{}, 5,
		); err != nil {
			return err
		}
		running, err = s.crawls.AmountOfRunningCrawls(tx)
		return err
	})

	if !slices.Equal(newest, []crawlid.CrawlID{third, second}) ||
		!slices.Equal(older, []crawlid.CrawlID{first}) ||
		!slices.Equal(newer, []crawlid.CrawlID{second, third}) ||
		!slices.Equal(succeeded, []crawlid.CrawlID{finished}) || running != 3 {
		t.Fatalf("newest %v older %v newer %v succeeded %v running %d",
			newest, older, newer, succeeded, running)
	}
}

func TestAStateChangeThePortRefusesFails(t *testing.T) {
	s := keptCrawls(t)
	crawlID := s.start(t, "o1")
	s.changes.refusing = true

	err := s.storage.Update(t.Context(), func(tx *vault.Txn) error {
		_, _, err := s.crawls.Pause(tx, crawlID)
		return err
	})

	if !errors.Is(err, errPortRefusing) {
		t.Fatalf("Pause = %v, want the port refusal", err)
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
	changeErr := s.storage.Update(t.Context(), func(tx *vault.Txn) error {
		_, _, err := s.crawls.Pause(tx, crawlID)
		return err
	})
	settleErr := s.storage.Update(t.Context(), func(tx *vault.Txn) error {
		return s.crawls.Settle(tx, crawlID)
	})
	listErr := s.storage.View(t.Context(), func(tx *vault.Txn) error {
		_, err := s.crawls.CrawlIDsOlderThan(
			tx, yacycrawlcontract.CrawlStateRunning, crawlid.CrawlID{}, 1,
		)
		return err
	})

	for name, err := range map[string]error{
		"start": startErr, "read": readErr, "change": changeErr, "settle": settleErr,
		"list": listErr,
	} {
		if !errors.Is(err, errEngineRefusing) {
			t.Errorf("%s = %v, want the refusal", name, err)
		}
	}
}
