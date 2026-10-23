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
	completed       []yacycrawlcontract.PageVisitResult
	dropped         []yacycrawlcontract.PageVisitDropReason
	unadmittedLinks []crawlid.CrawlID
}

func (r *recordedFailures) CompletedPageVisitNotCounted(
	_ context.Context,
	_ crawlid.CrawlID,
	result yacycrawlcontract.PageVisitResult,
	_ error,
) {
	r.completed = append(r.completed, result)
}

func (r *recordedFailures) DroppedPageVisitNotCounted(
	_ context.Context,
	_ crawlid.CrawlID,
	reason yacycrawlcontract.PageVisitDropReason,
	_ error,
) {
	r.dropped = append(r.dropped, reason)
}

func (r *recordedFailures) UnadmittedLinksNotCounted(
	_ context.Context,
	crawlID crawlid.CrawlID,
	_ error,
) {
	r.unadmittedLinks = append(r.unadmittedLinks, crawlID)
}

type progressScene struct {
	storage  *vault.Vault
	engine   *refusingEngine
	counts   *crawlprogress.Counts
	failures *recordedFailures
	counter  *crawlprogress.Counter
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
		counter: crawlprogress.NewCounter(storage, counts, failures),
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
	var progressByHost map[string]crawlprogress.Progress
	if err := s.storage.View(t.Context(), func(tx *vault.Txn) error {
		var err error
		if progress, err = s.counts.ProgressOf(tx, crawlID); err != nil {
			return err
		}
		progressByHost, err = s.counts.ProgressByHostOf(tx, crawlID)
		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
	return progress, progressByHost
}

func TestTheProgressOfACrawlCountsEndedVisitsAndUnadmittedLinks(t *testing.T) {
	s := newScene(t)

	s.counter.PendingPageVisitCompleted(t.Context(), pageVisitTo(t, knownCrawl, "http://a/1"),
		yacycrawlcontract.ResultPublished)
	s.counter.PendingPageVisitCompleted(t.Context(), pageVisitTo(t, knownCrawl, "http://b/1"),
		yacycrawlcontract.ResultPublished)
	s.counter.PendingPageVisitDropped(t.Context(), pageVisitTo(t, knownCrawl, "http://a/2"),
		yacycrawlcontract.DropReasonNotDue)
	s.counter.LinksUnadmitted(t.Context(), knownCrawl, 4)
	s.counter.PendingPageVisitReturned(t.Context(), pageVisitTo(t, knownCrawl, "http://a/3"), nil)
	s.counter.PendingPageVisitDeferred(t.Context(), pageVisitTo(t, knownCrawl, "http://a/3"), 0)
	s.counter.PendingPageVisitRetryScheduled(
		t.Context(),
		pageVisitTo(t, knownCrawl, "http://a/3"),
		0,
	)

	progress, progressByHost := s.progressOf(t, knownCrawl)
	if progress.Completed[yacycrawlcontract.ResultPublished] != 2 ||
		progress.Dropped[yacycrawlcontract.DropReasonNotDue] != 1 || progress.UnadmittedLinks != 4 {
		t.Fatalf("progress = %+v, want 2 published, 1 not due, 4 unadmitted", progress)
	}
	if hostA := progressByHost["a"]; hostA.Completed[yacycrawlcontract.ResultPublished] != 1 ||
		hostA.Dropped[yacycrawlcontract.DropReasonNotDue] != 1 || hostA.AmountOfURLs(3) != 5 {
		t.Fatalf("host a = %+v, want 1 published, 1 not due, 5 URLs with 3 pending", hostA)
	}
}

func TestACountTheStorageRefusesIsReported(t *testing.T) {
	s := newScene(t)
	s.engine.refusing = true

	s.counter.PendingPageVisitCompleted(t.Context(), pageVisitTo(t, knownCrawl, "http://a/1"),
		yacycrawlcontract.ResultPublished)
	s.counter.PendingPageVisitDropped(t.Context(), pageVisitTo(t, knownCrawl, "http://a/2"),
		yacycrawlcontract.DropReasonNotDue)
	s.counter.LinksUnadmitted(t.Context(), knownCrawl, 1)

	if len(s.failures.completed) != 1 ||
		s.failures.completed[0] != yacycrawlcontract.ResultPublished ||
		len(
			s.failures.dropped,
		) != 1 || s.failures.dropped[0] != yacycrawlcontract.DropReasonNotDue ||
		len(s.failures.unadmittedLinks) != 1 || s.failures.unadmittedLinks[0] != knownCrawl {
		t.Fatalf("failures = %+v, want each count reported", s.failures)
	}
}

func TestAForgottenCrawlCountsNothingAndOtherCrawlsKeepTheirCounts(t *testing.T) {
	s := newScene(t)
	otherCrawl := crawlid.CrawlID{2}
	for _, rawURL := range []string{"http://a/1", "http://b/1"} {
		s.counter.PendingPageVisitCompleted(t.Context(), pageVisitTo(t, knownCrawl, rawURL),
			yacycrawlcontract.ResultPublished)
	}
	s.counter.LinksUnadmitted(t.Context(), knownCrawl, 1)
	s.counter.PendingPageVisitCompleted(t.Context(), pageVisitTo(t, otherCrawl, "http://a/1"),
		yacycrawlcontract.ResultPublished)

	if err := s.storage.Update(t.Context(), func(tx *vault.Txn) error {
		return s.counts.ForgetCrawl(tx, knownCrawl)
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	progress, progressByHost := s.progressOf(t, knownCrawl)
	if progress.AmountOfURLs(0) != 0 || progress.UnadmittedLinks != 0 || len(progressByHost) != 0 {
		t.Fatalf("progress = %+v hosts %v, want nothing", progress, progressByHost)
	}
	if other, _ := s.progressOf(t, otherCrawl); other.AmountOfURLs(0) != 1 {
		t.Fatalf("other crawl = %+v, want its count kept", other)
	}
}
