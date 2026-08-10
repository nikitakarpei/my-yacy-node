package pagevisit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

type fakeFetch struct {
	mu         sync.Mutex
	outcome    FetchOutcome
	err        error
	gotVersion PageVersion
	sawFetch   bool
}

func (f *fakeFetch) Fetch(
	_ context.Context,
	_ string,
	knownVersion PageVersion,
) (FetchOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gotVersion = knownVersion
	f.sawFetch = true
	if f.err != nil {
		return FetchOutcome{}, f.err
	}
	return f.outcome, nil
}

type visitedCall struct {
	url     string
	version PageVersion
}

type fakeRecrawl struct {
	mu      sync.Mutex
	due     bool
	version PageVersion
	err     error

	visitedErr   error
	visitedCalls []visitedCall
}

func (f *fakeRecrawl) DecisionFor(
	context.Context,
	string,
) (RecrawlDecision, error) {
	if f.err != nil {
		return RecrawlDecision{}, f.err
	}
	return RecrawlDecision{Due: f.due, Version: f.version}, nil
}

func (f *fakeRecrawl) RecordVisit(
	_ context.Context,
	url string,
	version PageVersion,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.visitedCalls = append(f.visitedCalls, visitedCall{url: url, version: version})
	return f.visitedErr
}

func (f *fakeRecrawl) calls() []visitedCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]visitedCall(nil), f.visitedCalls...)
}

type fakeAbsorption struct {
	links    map[string][]string
	err      error
	disposal disposal.Reason
}

func (a *fakeAbsorption) Absorb(
	_ context.Context,
	page fetchedpage.Page,
) (pageabsorption.AbsorptionOutcome, error) {
	if a.err != nil {
		return pageabsorption.AbsorptionOutcome{}, a.err
	}
	return pageabsorption.AbsorptionOutcome{
		DiscoveredURLs: a.links[page.FinalURL],
		Disposal:       a.disposal,
	}, nil
}

type recordingObserver struct {
	mu       sync.Mutex
	refusals map[refusal.Demand]int
	fetched  int
}

func newObserver() *recordingObserver {
	return &recordingObserver{refusals: map[refusal.Demand]int{}}
}

func (*recordingObserver) FetchCompleted(time.Duration) {}

func (o *recordingObserver) PageFetched() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fetched++
}

func (o *recordingObserver) RefusalHonored(kind refusal.Demand) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.refusals[kind]++
}

type manualClock struct{ now time.Time }

func (c *manualClock) Now() time.Time { return c.now }

func (c *manualClock) Sleep(context.Context, time.Duration) error { return nil }

func fetchedOutcome() FetchOutcome {
	return FetchOutcome{
		Status: FetchSucceeded,
		Page: fetchedpage.Page{
			FinalURL:    "http://host/",
			ContentType: "text/html",
			Body:        []byte("x"),
		},
		Version: PageVersion{EntityTag: `"etag"`},
	}
}

func fetchOf(outcome FetchOutcome) *fakeFetch {
	return &fakeFetch{outcome: outcome}
}

func newVisitor(
	fetcher Fetcher,
	recrawl RecrawlRule,
	absorber pageabsorption.Absorber,
	observer VisitProgress,
	clock clock.Clock,
) *visitor {
	return &visitor{
		fetcher:  fetcher,
		recrawl:  recrawl,
		absorber: absorber,
		observer: observer,
		clock:    clock,
	}
}

func visitHost(t *testing.T, visitor Visitor) VisitOutcome {
	t.Helper()
	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != VisitCompleted {
		t.Fatalf("want concluded, got %v", outcome.Conclusion)
	}
	return outcome
}

func TestVisitAbsorbsFetchedPage(t *testing.T) {
	observer := newObserver()
	visitor := newVisitor(
		fetchOf(fetchedOutcome()),
		&fakeRecrawl{due: true},
		&fakeAbsorption{links: map[string][]string{
			"http://host/": {"http://host/next"},
		}},
		observer,
		&manualClock{},
	)

	outcome := visitHost(t, visitor)

	if observer.fetched != 1 {
		t.Fatalf("want one fetched page observed, got %d", observer.fetched)
	}

	if len(outcome.DiscoveredURLs) != 1 || outcome.DiscoveredURLs[0] != "http://host/next" {
		t.Fatalf("want discovered link, got %v", outcome.DiscoveredURLs)
	}
	if !outcome.Fetched {
		t.Fatal("an absorbed page counts as fetched")
	}
	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("published page should report no disposal, got %q", outcome.Disposal)
	}
}

func TestVisitAbsorptionErrorFails(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	visitor := newVisitor(
		fetchOf(fetchedOutcome()),
		recrawl,
		&fakeAbsorption{err: errors.New("absorb boom")},
		newObserver(),
		&manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err == nil {
		t.Fatal("absorption error should fail the visit")
	}
	if len(recrawl.calls()) != 0 {
		t.Fatalf("visited should not be recorded after absorb error, got %v", recrawl.calls())
	}
}

func TestVisitReportsNotAPageDisposal(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	visitor := newVisitor(
		fetchOf(FetchOutcome{Status: FetchNotAPage}),
		recrawl,
		&fakeAbsorption{},
		newObserver(),
		&manualClock{},
	)

	outcome := visitHost(t, visitor)

	if outcome.Disposal != disposal.NotAPage {
		t.Fatalf("want not-a-page disposal, got %q", outcome.Disposal)
	}
	if !outcome.Fetched {
		t.Fatal("a fetched body that is not a page still consumes the budget")
	}
	if len(recrawl.calls()) != 0 {
		t.Fatalf("visited should not be recorded for not-a-page, got %v", recrawl.calls())
	}
}

func TestVisitCeasesOnHTTPCease(t *testing.T) {
	observer := newObserver()
	recrawl := &fakeRecrawl{due: true}
	visitor := newVisitor(
		fetchOf(FetchOutcome{Status: FetchCeased}),
		recrawl,
		&fakeAbsorption{},
		observer,
		&manualClock{},
	)

	outcome := visitHost(t, visitor)

	if observer.refusals[refusal.Cease] != 1 {
		t.Fatalf("cease not honored: %v", observer.refusals)
	}
	if outcome.Disposal != disposal.Refused {
		t.Fatalf("want refused disposal, got %q", outcome.Disposal)
	}
	if outcome.Fetched {
		t.Fatal("a refused fetch returns no body and must not consume the budget")
	}
	if len(recrawl.calls()) != 1 {
		t.Fatalf("visited should be recorded on refusal so grace applies, got %v", recrawl.calls())
	}
}

func TestVisitReportsTransient(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	visitor := newVisitor(
		fetchOf(FetchOutcome{Status: FetchFailed}),
		recrawl,
		&fakeAbsorption{},
		newObserver(),
		&manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != VisitRetryable {
		t.Fatalf("want retryable, got %v", outcome.Conclusion)
	}
	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a transient failure must not dispose, got %q", outcome.Disposal)
	}
	if len(recrawl.calls()) != 0 {
		t.Fatalf("visited should not be recorded after failure, got %v", recrawl.calls())
	}
}

func TestVisitUnknownFetchStatusFails(t *testing.T) {
	visitor := newVisitor(
		fetchOf(FetchOutcome{Status: FetchStatus(99)}),
		&fakeRecrawl{due: true},
		&fakeAbsorption{},
		newObserver(),
		&manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err == nil {
		t.Fatal("an unknown fetch status should fail the visit, not retry silently")
	}
}

func TestVisitReportsDeferred(t *testing.T) {
	visitor := newVisitor(
		fetchOf(FetchOutcome{Status: FetchDeferred, DeferFor: time.Second}),
		&fakeRecrawl{due: true},
		&fakeAbsorption{},
		newObserver(),
		&manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != VisitDeferred || outcome.DeferFor != time.Second {
		t.Fatalf("want deferred for 1s, got %+v", outcome)
	}
}

func TestVisitFetchErrorFails(t *testing.T) {
	visitor := newVisitor(
		&fakeFetch{err: errors.New("boom")},
		&fakeRecrawl{due: true},
		&fakeAbsorption{},
		newObserver(),
		&manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err == nil {
		t.Fatal("fetch error should fail the visit")
	}
}

func TestVisitRecrawlDecisionErrorFails(t *testing.T) {
	visitor := newVisitor(
		&fakeFetch{}, &fakeRecrawl{err: errors.New("boom")}, &fakeAbsorption{},
		newObserver(), &manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err == nil {
		t.Fatal("recrawl decision error should fail the visit")
	}
}

func TestVisitReportsNotDueWithoutFetching(t *testing.T) {
	fetch := fetchOf(fetchedOutcome())
	visitor := newVisitor(
		fetch, &fakeRecrawl{due: false}, &fakeAbsorption{}, newObserver(), &manualClock{},
	)

	outcome := visitHost(t, visitor)

	if outcome.Disposal != disposal.NotDue {
		t.Fatalf("want a not-due disposal, got %q", outcome.Disposal)
	}
	if fetch.sawFetch {
		t.Fatal("fetch should not be attempted when not due")
	}
	if outcome.Fetched {
		t.Fatal("a skipped fetch must not consume the budget")
	}
}

func TestVisitPassesKnownVersionToFetcher(t *testing.T) {
	fetch := fetchOf(fetchedOutcome())
	known := PageVersion{EntityTag: `"stored-etag"`}
	visitor := newVisitor(
		fetch,
		&fakeRecrawl{due: true, version: known},
		&fakeAbsorption{},
		newObserver(),
		&manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err != nil {
		t.Fatalf("visit: %v", err)
	}
	if fetch.gotVersion != known {
		t.Fatalf("fetcher version = %+v, want %+v", fetch.gotVersion, known)
	}
}

func TestVisitRecordsVersionAfterSuccessfulAbsorb(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	visitor := newVisitor(
		fetchOf(fetchedOutcome()), recrawl, &fakeAbsorption{}, newObserver(), &manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err != nil {
		t.Fatalf("visit: %v", err)
	}
	calls := recrawl.calls()
	if len(calls) != 1 {
		t.Fatalf("want exactly one visited call, got %v", calls)
	}
	if calls[0].url != "http://host/" || calls[0].version.EntityTag != `"etag"` {
		t.Fatalf("visited call = %+v", calls[0])
	}
}

func TestVisitReportsAbsorptionDisposal(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	visitor := newVisitor(
		fetchOf(fetchedOutcome()),
		recrawl,
		&fakeAbsorption{disposal: disposal.Unextractable},
		newObserver(),
		&manualClock{},
	)

	outcome := visitHost(t, visitor)

	if outcome.Disposal != disposal.Unextractable {
		t.Fatalf("want the absorber's reason reported, got %q", outcome.Disposal)
	}
	if len(recrawl.calls()) != 1 {
		t.Fatalf("want the visit recorded regardless of publication, got %v", recrawl.calls())
	}
}

func TestVisitNotModifiedRecordsVersionWithoutAbsorbing(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	visitor := newVisitor(
		fetchOf(FetchOutcome{
			Status:  FetchNotModified,
			Version: PageVersion{EntityTag: `"same"`},
		}),
		recrawl,
		&fakeAbsorption{},
		newObserver(),
		&manualClock{},
	)

	outcome := visitHost(t, visitor)

	if outcome.Disposal != disposal.NotModified {
		t.Fatalf("want not-modified disposal, got %q", outcome.Disposal)
	}
	calls := recrawl.calls()
	if len(calls) != 1 || calls[0].version.EntityTag != `"same"` {
		t.Fatalf("want the version recorded once, got %v", calls)
	}
}

func TestVisitedErrorIsRecoverable(t *testing.T) {
	recrawl := &fakeRecrawl{due: true, visitedErr: errors.New("bucket down")}
	visitor := newVisitor(
		fetchOf(fetchedOutcome()), recrawl, &fakeAbsorption{}, newObserver(), &manualClock{},
	)

	visitHost(t, visitor)
}
