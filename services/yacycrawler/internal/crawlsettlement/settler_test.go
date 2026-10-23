package crawlsettlement_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlsettlement"
)

var (
	errSettlingEnds = errors.New("settling ends")
	errRefused      = errors.New("refused")
	emptyCrawl      = crawlid.CrawlID{1}
	busyCrawl       = crawlid.CrawlID{2}
)

type passes struct {
	left int
}

func (p *passes) Sleep(context.Context, time.Duration) error {
	p.left--
	if p.left <= 0 {
		return errSettlingEnds
	}
	return nil
}

type keptCrawls struct {
	unfinished       []crawlid.CrawlID
	settled          []crawlid.CrawlID
	listRefused      bool
	settleRefusals   int
	settleRefusalsOf crawlid.CrawlID
}

func (c *keptCrawls) UnfinishedCrawlIDs(*vault.Txn) ([]crawlid.CrawlID, error) {
	if c.listRefused {
		return nil, errRefused
	}
	return slices.Clone(c.unfinished), nil
}

func (c *keptCrawls) Settle(_ *vault.Txn, crawlID crawlid.CrawlID) error {
	if crawlID == c.settleRefusalsOf && c.settleRefusals > 0 {
		c.settleRefusals--
		return errRefused
	}
	c.settled = append(c.settled, crawlID)
	c.unfinished = slices.DeleteFunc(c.unfinished, func(id crawlid.CrawlID) bool {
		return id == crawlID
	})
	return nil
}

type pendingCounts struct {
	amountsOfCrawl map[crawlid.CrawlID][]int
	refused        bool
}

func (p *pendingCounts) AmountPendingOf(_ *vault.Txn, crawlID crawlid.CrawlID) (int, error) {
	if p.refused {
		return 0, errRefused
	}
	amounts := p.amountsOfCrawl[crawlID]
	if len(amounts) == 0 {
		return 0, nil
	}
	if len(amounts) > 1 {
		p.amountsOfCrawl[crawlID] = amounts[1:]
	}
	return amounts[0], nil
}

type reportedFailures struct {
	unread     []error
	notSettled []crawlid.CrawlID
}

func (r *reportedFailures) UnfinishedCrawlsNotRead(_ context.Context, err error) {
	r.unread = append(r.unread, err)
}

func (r *reportedFailures) CrawlNotSettled(_ context.Context, crawlID crawlid.CrawlID, _ error) {
	r.notSettled = append(r.notSettled, crawlID)
}

type scene struct {
	crawls   *keptCrawls
	pending  *pendingCounts
	failures *reportedFailures
}

func newScene(unfinished ...crawlid.CrawlID) scene {
	return scene{
		crawls:   &keptCrawls{unfinished: unfinished},
		pending:  &pendingCounts{amountsOfCrawl: map[crawlid.CrawlID][]int{}},
		failures: &reportedFailures{},
	}
}

func (s scene) settleFor(t *testing.T, amountOfPasses int) {
	t.Helper()
	storage, err := vault.New(memoryvault.OpenEngine(0), nil, nil, vaultenginetest.FrozenClock{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	settler := crawlsettlement.NewSettler(
		storage, s.crawls, s.pending, &passes{left: amountOfPasses}, s.failures,
	)
	if err := settler.Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestACrawlWithNothingPendingSettlesAndOneWithPendingVisitsStaysUnfinished(t *testing.T) {
	s := newScene(emptyCrawl, busyCrawl)
	s.pending.amountsOfCrawl[busyCrawl] = []int{3}

	s.settleFor(t, 1)

	if !slices.Equal(s.crawls.settled, []crawlid.CrawlID{emptyCrawl}) {
		t.Fatalf("settled %v, want only the crawl with nothing pending", s.crawls.settled)
	}
}

func TestACrawlThatTakesAVisitBeforeItSettlesStaysUnfinished(t *testing.T) {
	s := newScene(busyCrawl)
	s.pending.amountsOfCrawl[busyCrawl] = []int{0, 1}

	s.settleFor(t, 1)

	if len(s.crawls.settled) != 0 {
		t.Fatalf("settled %v, want the crawl that took a visit left unfinished", s.crawls.settled)
	}
}

func TestACrawlThatCannotSettleIsReportedAndSettlesOnALaterPass(t *testing.T) {
	s := newScene(emptyCrawl)
	s.crawls.settleRefusalsOf, s.crawls.settleRefusals = emptyCrawl, 1

	s.settleFor(t, 2)

	if !slices.Equal(s.failures.notSettled, []crawlid.CrawlID{emptyCrawl}) ||
		!slices.Equal(s.crawls.settled, []crawlid.CrawlID{emptyCrawl}) {
		t.Fatalf("not settled %v, settled %v; want one failure, then the crawl settled",
			s.failures.notSettled, s.crawls.settled)
	}
}

func TestUnfinishedCrawlsThatCannotBeReadAreReported(t *testing.T) {
	s := newScene(emptyCrawl)
	s.crawls.listRefused = true

	s.settleFor(t, 1)

	if len(s.failures.unread) != 1 || len(s.crawls.settled) != 0 {
		t.Fatalf("unread %v, settled %v; want the failure reported and nothing settled",
			s.failures.unread, s.crawls.settled)
	}
}

func TestACrawlWhosePendingVisitsCannotBeCountedIsReportedAndLeftUnfinished(t *testing.T) {
	s := newScene(emptyCrawl)
	s.pending.refused = true

	s.settleFor(t, 1)

	if !slices.Equal(s.failures.notSettled, []crawlid.CrawlID{emptyCrawl}) ||
		len(s.crawls.settled) != 0 {
		t.Fatalf("not settled %v, settled %v; want the failure reported and nothing settled",
			s.failures.notSettled, s.crawls.settled)
	}
}
