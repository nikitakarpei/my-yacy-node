package crawlprogress_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlprogress"
)

var (
	knownCrawl        = crawlid.CrawlID{1}
	errEngineRefusing = errors.New("engine refuses")
)

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

type recordedFailures struct {
	counts []string
}

func (r *recordedFailures) CrawlProgressCountFailed(
	_ context.Context,
	_ crawlid.CrawlID,
	count string,
	_ error,
) {
	r.counts = append(r.counts, count)
}

type progressScene struct {
	storage  *vault.Vault
	engine   *refusingEngine
	counts   *crawlprogress.Counts
	failures *recordedFailures
	counting *crawlprogress.Counting
}

func newScene(t *testing.T) progressScene {
	t.Helper()
	engine := &refusingEngine{
		Engine: vaultenginetest.EngineRepeatingWrites(memoryvault.OpenEngine(0)),
	}
	storage, err := vault.New(engine, nil, nil, vaultenginetest.FrozenClock{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	counts, err := crawlprogress.Open(storage)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	failures := &recordedFailures{}
	return progressScene{
		storage: storage, engine: engine, counts: counts, failures: failures,
		counting: crawlprogress.NewCounting(storage, counts, failures),
	}
}

func pageVisitTo(
	t *testing.T,
	crawlID crawlid.CrawlID,
	rawURL string,
) pendingpagevisit.PendingPageVisit {
	t.Helper()
	return pendingpagevisit.PendingPageVisit{
		CrawlID: crawlID,
		URL:     canonicalurltest.CanonicalURLOf(t, rawURL),
	}
}

func (s progressScene) progressOf(t *testing.T, crawlID crawlid.CrawlID) (
	crawlprogress.Progress,
	map[string]crawlprogress.Progress,
) {
	t.Helper()
	var progress crawlprogress.Progress
	var progressOfHosts map[string]crawlprogress.Progress
	if err := s.storage.View(t.Context(), func(tx *vault.Txn) error {
		var err error
		if progress, err = s.counts.ProgressOf(tx, crawlID); err != nil {
			return err
		}
		progressOfHosts, err = s.counts.ProgressOfHostsOf(tx, crawlID)
		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
	return progress, progressOfHosts
}

func TestTheProgressOfACrawlCountsSettledVisitsAndUnadmittedLinks(t *testing.T) {
	s := newScene(t)

	s.counting.PendingPageVisitCompleted(t.Context(), pageVisitTo(t, knownCrawl, "http://a/1"),
		yacycrawlcontract.ResultPublished)
	s.counting.PendingPageVisitCompleted(t.Context(), pageVisitTo(t, knownCrawl, "http://b/1"),
		yacycrawlcontract.ResultPublished)
	s.counting.PendingPageVisitDropped(t.Context(), pageVisitTo(t, knownCrawl, "http://a/2"),
		yacycrawlcontract.DropReasonNotDue)
	s.counting.LinksUnadmitted(t.Context(), knownCrawl, 4)
	s.counting.PendingPageVisitReturned(t.Context(), pageVisitTo(t, knownCrawl, "http://a/3"), nil)
	s.counting.PendingPageVisitDeferred(t.Context(), pageVisitTo(t, knownCrawl, "http://a/3"), 0)
	s.counting.PendingPageVisitRetryScheduled(
		t.Context(),
		pageVisitTo(t, knownCrawl, "http://a/3"),
		0,
	)

	progress, progressOfHosts := s.progressOf(t, knownCrawl)
	if progress.Completed[yacycrawlcontract.ResultPublished] != 2 ||
		progress.Dropped[yacycrawlcontract.DropReasonNotDue] != 1 || progress.UnadmittedLinks != 4 {
		t.Fatalf("progress = %+v, want 2 published, 1 not due, 4 unadmitted", progress)
	}
	if hostA := progressOfHosts["a"]; hostA.Completed[yacycrawlcontract.ResultPublished] != 1 ||
		hostA.Dropped[yacycrawlcontract.DropReasonNotDue] != 1 || hostA.AmountOfURLs(3) != 5 {
		t.Fatalf("host a = %+v, want 1 published, 1 not due, 5 URLs with 3 owed", hostA)
	}
}

func TestACountTheStorageRefusesIsReported(t *testing.T) {
	s := newScene(t)
	s.engine.refusing = true

	s.counting.PendingPageVisitDropped(t.Context(), pageVisitTo(t, knownCrawl, "http://a/1"),
		yacycrawlcontract.DropReasonNotDue)

	if len(s.failures.counts) != 1 || s.failures.counts[0] != "dropped/not-due" {
		t.Fatalf("failures = %v, want the dropped count reported", s.failures.counts)
	}
}

func TestAForgottenCrawlCountsNothingAndOtherCrawlsKeepTheirCounts(t *testing.T) {
	s := newScene(t)
	otherCrawl := crawlid.CrawlID{2}
	for _, rawURL := range []string{"http://a/1", "http://b/1"} {
		s.counting.PendingPageVisitCompleted(t.Context(), pageVisitTo(t, knownCrawl, rawURL),
			yacycrawlcontract.ResultPublished)
	}
	s.counting.LinksUnadmitted(t.Context(), knownCrawl, 1)
	s.counting.PendingPageVisitCompleted(t.Context(), pageVisitTo(t, otherCrawl, "http://a/1"),
		yacycrawlcontract.ResultPublished)

	if err := s.storage.Update(t.Context(), func(tx *vault.Txn) error {
		return s.counts.ForgetCrawl(tx, knownCrawl)
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	progress, progressOfHosts := s.progressOf(t, knownCrawl)
	if progress.AmountOfURLs(0) != 0 || progress.UnadmittedLinks != 0 || len(progressOfHosts) != 0 {
		t.Fatalf("progress = %+v hosts %v, want nothing", progress, progressOfHosts)
	}
	if other, _ := s.progressOf(t, otherCrawl); other.AmountOfURLs(0) != 1 {
		t.Fatalf("other crawl = %+v, want its count kept", other)
	}
}
