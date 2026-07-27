package pagevisit_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

type fakeFetch struct {
	mu            sync.Mutex
	outcomes      map[string][]pagevisit.FetchOutcome
	err           error
	gotValidators pagevisit.Revisit
	sawValidators bool
}

func (f *fakeFetch) Fetch(
	_ context.Context,
	rawURL string,
	validators pagevisit.Revisit,
) (pagevisit.FetchOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gotValidators = validators
	f.sawValidators = true
	if f.err != nil {
		return pagevisit.FetchOutcome{}, f.err
	}
	queue := f.outcomes[rawURL]
	if len(queue) == 0 {
		return pagevisit.FetchOutcome{Status: pagevisit.FetchNotAPage}, nil
	}
	outcome := queue[0]
	if len(queue) > 1 {
		f.outcomes[rawURL] = queue[1:]
	}
	return outcome, nil
}

type visitedCall struct {
	url        string
	validators pagevisit.Revisit
}

type fakeRecrawl struct {
	mu      sync.Mutex
	due     bool
	revisit pagevisit.Revisit
	err     error

	visitedErr   error
	visitedCalls []visitedCall
}

func (f *fakeRecrawl) Revisit(context.Context, string) (pagevisit.Revisit, error) {
	if f.err != nil {
		return pagevisit.Revisit{}, f.err
	}
	revisit := f.revisit
	revisit.Due = f.due
	return revisit, nil
}

func (f *fakeRecrawl) Visited(_ context.Context, url string, validators pagevisit.Revisit) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.visitedCalls = append(f.visitedCalls, visitedCall{url: url, validators: validators})
	return f.visitedErr
}

func (f *fakeRecrawl) calls() []visitedCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]visitedCall(nil), f.visitedCalls...)
}

type fakeAbsorption struct {
	links       map[string][]string
	err         error
	unpublished bool
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
		Published:      !a.unpublished,
	}, nil
}

type fakeDisposedPages struct {
	mu   sync.Mutex
	urls []string
	err  error
}

func (d *fakeDisposedPages) Record(_ context.Context, url string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.urls = append(d.urls, url)
	return d.err
}

func (d *fakeDisposedPages) calls() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.urls...)
}

type recordingObserver struct {
	mu       sync.Mutex
	disposed map[disposal.Reason]int
	refusals map[refusal.Demand]int
	fetched  int
}

func newObserver() *recordingObserver {
	return &recordingObserver{
		disposed: map[disposal.Reason]int{},
		refusals: map[refusal.Demand]int{},
	}
}

func (*recordingObserver) FetchCompleted(time.Duration) {}

func (o *recordingObserver) PageFetched() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fetched++
}

func (o *recordingObserver) PageDisposed(reason disposal.Reason) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.disposed[reason]++
}

func (o *recordingObserver) RefusalHonored(kind refusal.Demand) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.refusals[kind]++
}

type manualClock struct{ now time.Time }

func (c *manualClock) Now() time.Time { return c.now }

func (c *manualClock) Sleep(context.Context, time.Duration) error { return nil }

func fetchedOutcome() pagevisit.FetchOutcome {
	return pagevisit.FetchOutcome{
		Status: pagevisit.FetchSucceeded,
		Page: fetchedpage.Page{
			FinalURL:    "http://host/",
			ContentType: "text/html",
			Body:        []byte("x"),
		},
		Validators: pagevisit.Revisit{EntityTag: `"etag"`},
	}
}

func TestVisitAbsorbsFetchedPage(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	disposed := &fakeDisposedPages{}
	visitor := pagevisit.New(
		fetch,
		&fakeRecrawl{due: true},
		&fakeAbsorption{links: map[string][]string{
			"http://host/": {"http://host/next"},
		}},
		disposed,
		newObserver(),
		&manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != pagevisit.VisitCompleted {
		t.Fatalf("want concluded, got %v", outcome.Conclusion)
	}
	if len(outcome.DiscoveredURLs) != 1 || outcome.DiscoveredURLs[0] != "http://host/next" {
		t.Fatalf("want discovered link, got %v", outcome.DiscoveredURLs)
	}
	if len(disposed.calls()) != 0 {
		t.Fatalf("published page should not be recorded disposed, got %v", disposed.calls())
	}
}

func TestVisitAbsorptionErrorFails(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	recrawl := &fakeRecrawl{due: true}
	visitor := pagevisit.New(
		fetch,
		recrawl,
		&fakeAbsorption{err: errors.New("absorb boom")},
		&fakeDisposedPages{},
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

func TestVisitDisposesNotAPage(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {{Status: pagevisit.FetchNotAPage}},
	}}
	observer := newObserver()
	recrawl := &fakeRecrawl{due: true}
	disposed := &fakeDisposedPages{}
	visitor := pagevisit.New(
		fetch,
		recrawl,
		&fakeAbsorption{},
		disposed,
		observer,
		&manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != pagevisit.VisitCompleted {
		t.Fatalf("want concluded, got %v", outcome.Conclusion)
	}
	if observer.disposed[disposal.NotAPage] != 1 {
		t.Fatalf("want not-a-page disposal, got %v", observer.disposed)
	}
	if len(recrawl.calls()) != 0 {
		t.Fatalf("visited should not be recorded for not-a-page, got %v", recrawl.calls())
	}
	if len(disposed.calls()) != 1 {
		t.Fatalf("want disposed page recorded once, got %v", disposed.calls())
	}
}

func TestVisitCeasesOnHTTPCease(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {{Status: pagevisit.FetchCeased}},
	}}
	observer := newObserver()
	recrawl := &fakeRecrawl{due: true}
	disposed := &fakeDisposedPages{}
	visitor := pagevisit.New(
		fetch,
		recrawl,
		&fakeAbsorption{},
		disposed,
		observer,
		&manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != pagevisit.VisitCompleted {
		t.Fatalf("want concluded, got %v", outcome.Conclusion)
	}
	if observer.refusals[refusal.Cease] != 1 {
		t.Fatalf("cease not honored: %v", observer.refusals)
	}
	if len(recrawl.calls()) != 1 {
		t.Fatalf("visited should be recorded on refusal so grace applies, got %v", recrawl.calls())
	}
	if len(disposed.calls()) != 1 {
		t.Fatalf("want disposed page recorded once, got %v", disposed.calls())
	}
}

func TestVisitReportsTransient(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {{Status: pagevisit.FetchFailed}},
	}}
	recrawl := &fakeRecrawl{due: true}
	disposed := &fakeDisposedPages{}
	visitor := pagevisit.New(
		fetch,
		recrawl,
		&fakeAbsorption{},
		disposed,
		newObserver(),
		&manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != pagevisit.VisitRetryable {
		t.Fatalf("want retryable, got %v", outcome.Conclusion)
	}
	if len(recrawl.calls()) != 0 {
		t.Fatalf("visited should not be recorded after failure, got %v", recrawl.calls())
	}
	if len(disposed.calls()) != 0 {
		t.Fatalf("a transient failure must not produce a fast negative, got %v", disposed.calls())
	}
}

func TestVisitReportsDeferred(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {{Status: pagevisit.FetchDeferred, DeferFor: time.Second}},
	}}
	disposed := &fakeDisposedPages{}
	visitor := pagevisit.New(
		fetch,
		&fakeRecrawl{due: true},
		&fakeAbsorption{},
		disposed,
		newObserver(),
		&manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != pagevisit.VisitDeferred || outcome.DeferFor != time.Second {
		t.Fatalf("want deferred for 1s, got %+v", outcome)
	}
}

func TestVisitFetchErrorFails(t *testing.T) {
	fetch := &fakeFetch{err: errors.New("boom")}
	visitor := pagevisit.New(
		fetch,
		&fakeRecrawl{due: true},
		&fakeAbsorption{},
		&fakeDisposedPages{},
		newObserver(),
		&manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err == nil {
		t.Fatal("fetch error should fail the visit")
	}
}

func TestVisitSkipsFetchWhenNotDue(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	absorption := &fakeAbsorption{}
	visitor := pagevisit.New(
		fetch,
		&fakeRecrawl{due: false},
		absorption,
		&fakeDisposedPages{},
		newObserver(),
		&manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != pagevisit.VisitCompleted {
		t.Fatalf("want concluded, got %v", outcome.Conclusion)
	}
	if fetch.sawValidators {
		t.Fatal("fetch should not be attempted when not due")
	}
}

func TestVisitRecrawlDecisionErrorFails(t *testing.T) {
	visitor := pagevisit.New(
		&fakeFetch{}, &fakeRecrawl{err: errors.New("boom")}, &fakeAbsorption{},
		&fakeDisposedPages{}, newObserver(), &manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err == nil {
		t.Fatal("recrawl decision error should fail the visit")
	}
}

func TestVisitDisposesPageNotDue(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	observer := newObserver()
	disposed := &fakeDisposedPages{}
	visitor := pagevisit.New(
		fetch, &fakeRecrawl{due: false}, &fakeAbsorption{}, disposed, observer, &manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != pagevisit.VisitCompleted {
		t.Fatalf("want concluded, got %v", outcome.Conclusion)
	}
	if observer.disposed[disposal.NotDue] != 1 {
		t.Fatalf("want a not-due disposal recorded, got %v", observer.disposed)
	}
	if len(disposed.calls()) != 1 {
		t.Fatalf("want disposed page recorded once, got %v", disposed.calls())
	}
}

func TestVisitPassesValidatorsToFetcher(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	stored := pagevisit.Revisit{EntityTag: `"stored-etag"`}
	visitor := pagevisit.New(
		fetch,
		&fakeRecrawl{due: true, revisit: stored},
		&fakeAbsorption{},
		&fakeDisposedPages{},
		newObserver(),
		&manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err != nil {
		t.Fatalf("visit: %v", err)
	}
	want := stored
	want.Due = true
	if fetch.gotValidators != want {
		t.Fatalf("fetcher validators = %+v, want %+v", fetch.gotValidators, want)
	}
}

func TestVisitRecordsValidatorsAfterSuccessfulAbsorb(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	recrawl := &fakeRecrawl{due: true}
	disposed := &fakeDisposedPages{}
	visitor := pagevisit.New(
		fetch, recrawl, &fakeAbsorption{}, disposed, newObserver(), &manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err != nil {
		t.Fatalf("visit: %v", err)
	}
	calls := recrawl.calls()
	if len(calls) != 1 {
		t.Fatalf("want exactly one visited call, got %v", calls)
	}
	if calls[0].url != "http://host/" || calls[0].validators.EntityTag != `"etag"` {
		t.Fatalf("visited call = %+v", calls[0])
	}
	if len(disposed.calls()) != 0 {
		t.Fatalf("published page should not be recorded disposed, got %v", disposed.calls())
	}
}

func TestVisitAbsorbUnpublishedRecordsDisposal(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	recrawl := &fakeRecrawl{due: true}
	disposed := &fakeDisposedPages{}
	visitor := pagevisit.New(
		fetch, recrawl, &fakeAbsorption{unpublished: true}, disposed, newObserver(), &manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err != nil {
		t.Fatalf("visit: %v", err)
	}
	if len(recrawl.calls()) != 1 {
		t.Fatalf("want the visit recorded regardless of publication, got %v", recrawl.calls())
	}
	if len(disposed.calls()) != 1 {
		t.Fatalf("want disposed page recorded once, got %v", disposed.calls())
	}
}

func TestVisitNotModifiedDisposesAndRecordsWithoutAbsorbing(t *testing.T) {
	notModified := pagevisit.FetchOutcome{
		Status:     pagevisit.FetchNotModified,
		Validators: pagevisit.Revisit{EntityTag: `"same"`},
	}
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {notModified},
	}}
	observer := newObserver()
	recrawl := &fakeRecrawl{due: true}
	absorption := &fakeAbsorption{}
	disposed := &fakeDisposedPages{}
	visitor := pagevisit.New(fetch, recrawl, absorption, disposed, observer, &manualClock{})

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != pagevisit.VisitCompleted {
		t.Fatalf("want concluded, got %v", outcome.Conclusion)
	}
	if observer.disposed[disposal.NotModified] != 1 {
		t.Fatalf("want not-modified disposal, got %v", observer.disposed)
	}
	calls := recrawl.calls()
	if len(calls) != 1 || calls[0].validators.EntityTag != `"same"` {
		t.Fatalf("want validators recorded once, got %v", calls)
	}
	if len(disposed.calls()) != 1 {
		t.Fatalf("want disposed page recorded once, got %v", disposed.calls())
	}
}

func TestVisitedErrorIsRecoverable(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	recrawl := &fakeRecrawl{due: true, visitedErr: errors.New("bucket down")}
	visitor := pagevisit.New(
		fetch, recrawl, &fakeAbsorption{}, &fakeDisposedPages{}, newObserver(), &manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visited error should not fail the visit: %v", err)
	}
	if outcome.Conclusion != pagevisit.VisitCompleted {
		t.Fatalf("want concluded, got %v", outcome.Conclusion)
	}
}
