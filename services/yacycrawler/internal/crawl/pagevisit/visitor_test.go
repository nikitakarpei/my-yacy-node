package pagevisit_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

type fakeFetch struct {
	mu       sync.Mutex
	outcomes map[string][]pagevisit.FetchOutcome
	err      error
}

func (f *fakeFetch) Fetch(_ context.Context, rawURL string) (pagevisit.FetchOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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

type fakeRecrawl struct {
	due bool
	err error
}

func (f fakeRecrawl) Due(context.Context, string) (bool, error) { return f.due, f.err }

type fakeAbsorption struct {
	links map[string][]string
	err   error
}

func (a *fakeAbsorption) Absorb(
	_ context.Context,
	page fetchedpage.Page,
) ([]string, error) {
	if a.err != nil {
		return nil, a.err
	}
	return a.links[page.FinalURL], nil
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
	}
}

func TestVisitAbsorbsFetchedPage(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	visitor := pagevisit.New(
		fetch,
		fakeRecrawl{due: true},
		&fakeAbsorption{links: map[string][]string{
			"http://host/": {"http://host/next"},
		}},
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
}

func TestVisitAbsorptionErrorFails(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	visitor := pagevisit.New(
		fetch,
		fakeRecrawl{due: true},
		&fakeAbsorption{err: errors.New("absorb boom")},
		newObserver(),
		&manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err == nil {
		t.Fatal("absorption error should fail the visit")
	}
}

func TestVisitDisposesNotAPage(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {{Status: pagevisit.FetchNotAPage}},
	}}
	observer := newObserver()
	visitor := pagevisit.New(
		fetch,
		fakeRecrawl{due: true},
		&fakeAbsorption{},
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
}

func TestVisitCeasesOnHTTPCease(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {{Status: pagevisit.FetchCeased}},
	}}
	observer := newObserver()
	visitor := pagevisit.New(
		fetch,
		fakeRecrawl{due: true},
		&fakeAbsorption{},
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
}

func TestVisitReportsTransient(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {{Status: pagevisit.FetchFailed}},
	}}
	visitor := pagevisit.New(
		fetch,
		fakeRecrawl{due: true},
		&fakeAbsorption{},
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
}

func TestVisitReportsDeferred(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {{Status: pagevisit.FetchDeferred, DeferFor: time.Second}},
	}}
	visitor := pagevisit.New(
		fetch,
		fakeRecrawl{due: true},
		&fakeAbsorption{},
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
		fakeRecrawl{due: true},
		&fakeAbsorption{},
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
		fakeRecrawl{due: false},
		absorption,
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
}

func TestVisitRecrawlDecisionErrorFails(t *testing.T) {
	visitor := pagevisit.New(
		&fakeFetch{}, fakeRecrawl{err: errors.New("boom")}, &fakeAbsorption{},
		newObserver(), &manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err == nil {
		t.Fatal("recrawl decision error should fail the visit")
	}
}

type notDuePolicy struct{}

func (notDuePolicy) Due(context.Context, string) (bool, error) { return false, nil }

func TestVisitDisposesPageNotDue(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]pagevisit.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	observer := newObserver()
	visitor := pagevisit.New(
		fetch, notDuePolicy{}, &fakeAbsorption{}, observer, &manualClock{},
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
}
