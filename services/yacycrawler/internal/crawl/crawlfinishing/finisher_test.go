package crawlfinishing_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlfinishing"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

var (
	errFinishingStops = errors.New("finishing stops")
	errRefused        = errors.New("refused")
	emptyCrawl        = crawlid.CrawlID{1}
	busyCrawl         = crawlid.CrawlID{2}
)

type passes struct {
	left int
}

func (p *passes) Sleep(context.Context, time.Duration) error {
	p.left--
	if p.left <= 0 {
		return errFinishingStops
	}
	return nil
}

type keptCrawls struct {
	unfinished       []crawlid.CrawlID
	finished         []crawlid.CrawlID
	listRefused      bool
	finishRefusals   int
	finishRefusalsOf crawlid.CrawlID
}

func (c *keptCrawls) UnfinishedCrawlIDs(*vault.Txn) ([]crawlid.CrawlID, error) {
	if c.listRefused {
		return nil, errRefused
	}
	return slices.Clone(c.unfinished), nil
}

func (c *keptCrawls) Finish(_ *vault.Txn, crawlID crawlid.CrawlID) error {
	if crawlID == c.finishRefusalsOf && c.finishRefusals > 0 {
		c.finishRefusals--
		return errRefused
	}
	c.finished = append(c.finished, crawlID)
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
	unread      []error
	notFinished []crawlid.CrawlID
}

func (r *reportedFailures) UnfinishedCrawlsNotRead(_ context.Context, err error) {
	r.unread = append(r.unread, err)
}

func (r *reportedFailures) CrawlNotFinished(_ context.Context, crawlID crawlid.CrawlID, _ error) {
	r.notFinished = append(r.notFinished, crawlID)
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

func (s scene) finishFor(t *testing.T, amountOfPasses int) {
	t.Helper()
	storage, err := vault.New(memoryvault.OpenEngine(0), nil, nil, vaultenginetest.FrozenClock{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	finisher := crawlfinishing.NewFinisher(
		storage, s.crawls, s.pending, &passes{left: amountOfPasses}, s.failures,
	)
	if err := finisher.Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestACrawlWithNothingPendingFinishesAndOneWithPendingVisitsStaysUnfinished(t *testing.T) {
	s := newScene(emptyCrawl, busyCrawl)
	s.pending.amountsOfCrawl[busyCrawl] = []int{3}

	s.finishFor(t, 1)

	if !slices.Equal(s.crawls.finished, []crawlid.CrawlID{emptyCrawl}) {
		t.Fatalf("finished %v, want only the crawl with nothing pending", s.crawls.finished)
	}
}

func TestACrawlThatTakesAVisitBeforeItFinishesStaysUnfinished(t *testing.T) {
	s := newScene(busyCrawl)
	s.pending.amountsOfCrawl[busyCrawl] = []int{0, 1}

	s.finishFor(t, 1)

	if len(s.crawls.finished) != 0 {
		t.Fatalf("finished %v, want the crawl that took a visit left unfinished", s.crawls.finished)
	}
}

func TestACrawlThatCannotFinishIsReportedAndFinishesOnALaterPass(t *testing.T) {
	s := newScene(emptyCrawl)
	s.crawls.finishRefusalsOf, s.crawls.finishRefusals = emptyCrawl, 1

	s.finishFor(t, 2)

	if !slices.Equal(s.failures.notFinished, []crawlid.CrawlID{emptyCrawl}) ||
		!slices.Equal(s.crawls.finished, []crawlid.CrawlID{emptyCrawl}) {
		t.Fatalf("not finished %v, finished %v; want one failure, then the crawl finished",
			s.failures.notFinished, s.crawls.finished)
	}
}

func TestUnfinishedCrawlsThatCannotBeReadAreReported(t *testing.T) {
	s := newScene(emptyCrawl)
	s.crawls.listRefused = true

	s.finishFor(t, 1)

	if len(s.failures.unread) != 1 || len(s.crawls.finished) != 0 {
		t.Fatalf("unread %v, finished %v; want the failure reported and nothing finished",
			s.failures.unread, s.crawls.finished)
	}
}

func TestACrawlWhosePendingVisitsCannotBeCountedIsReportedAndLeftUnfinished(t *testing.T) {
	s := newScene(emptyCrawl)
	s.pending.refused = true

	s.finishFor(t, 1)

	if !slices.Equal(s.failures.notFinished, []crawlid.CrawlID{emptyCrawl}) ||
		len(s.crawls.finished) != 0 {
		t.Fatalf("not finished %v, finished %v; want the failure reported and nothing finished",
			s.failures.notFinished, s.crawls.finished)
	}
}
