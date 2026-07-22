package pagevisit_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisit"
)

type fakeFetch struct {
	mu       sync.Mutex
	outcomes map[string][]crawlcapability.FetchOutcome
	err      error
}

func (f *fakeFetch) Fetch(_ context.Context, rawURL string) (crawlcapability.FetchOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return crawlcapability.FetchOutcome{}, f.err
	}
	queue := f.outcomes[rawURL]
	if len(queue) == 0 {
		return crawlcapability.FetchOutcome{Status: crawlcapability.FetchNotAPage}, nil
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
	outcome crawlcapability.FetchOutcome,
) ([]string, error) {
	if a.err != nil {
		return nil, a.err
	}
	return a.links[outcome.FinalURL], nil
}

type recordingObserver struct {
	mu       sync.Mutex
	disposed map[string]int
	refusals map[string]int
	fetched  int
}

func newObserver() *recordingObserver {
	return &recordingObserver{disposed: map[string]int{}, refusals: map[string]int{}}
}

func (*recordingObserver) OrderReceived()              {}
func (*recordingObserver) OrderRedelivered()           {}
func (*recordingObserver) OrderCompleted()             {}
func (*recordingObserver) PagePublished(string)        {}
func (*recordingObserver) PublicationWaited()          {}
func (*recordingObserver) FetchObserved(time.Duration) {}
func (*recordingObserver) BudgetExhausted()            {}

func (o *recordingObserver) PageFetched() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fetched++
}

func (o *recordingObserver) PageDisposed(reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.disposed[reason]++
}

func (o *recordingObserver) RefusalHonored(kind string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.refusals[kind]++
}

type manualClock struct{ now time.Time }

func (c *manualClock) Now() time.Time { return c.now }

func (c *manualClock) Sleep(context.Context, time.Duration) error { return nil }

func fetchedOutcome() crawlcapability.FetchOutcome {
	return crawlcapability.FetchOutcome{
		Status: crawlcapability.FetchSucceeded, FinalURL: "http://host/", ContentType: "text/html",
		Body: []byte("x"),
	}
}

func TestVisitAbsorbsFetchedPage(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	visitor := pagevisit.NewPageVisit(
		fetch, pagevisit.AlwaysDue{}, &fakeAbsorption{links: map[string][]string{
			"http://host/": {"http://host/next"},
		}}, newObserver(), &manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Classification != pagevisit.Succeeded {
		t.Fatalf("want succeeded, got %v", outcome.Classification)
	}
	if len(outcome.DiscoveredURLs) != 1 || outcome.DiscoveredURLs[0] != "http://host/next" {
		t.Fatalf("want discovered link, got %v", outcome.DiscoveredURLs)
	}
}

func TestVisitAbsorptionErrorFails(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	visitor := pagevisit.NewPageVisit(
		fetch, pagevisit.AlwaysDue{}, &fakeAbsorption{err: errors.New("absorb boom")},
		newObserver(), &manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err == nil {
		t.Fatal("absorption error should fail the visit")
	}
}

func TestVisitDisposesNotAPage(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {{Status: crawlcapability.FetchNotAPage}},
	}}
	observer := newObserver()
	visitor := pagevisit.NewPageVisit(
		fetch, pagevisit.AlwaysDue{}, &fakeAbsorption{}, observer, &manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Classification != pagevisit.NotAPage {
		t.Fatalf("want not-a-page, got %v", outcome.Classification)
	}
	if observer.disposed[crawlcapability.DisposalFetchFailed] != 1 {
		t.Fatalf("want fetch-failed disposal, got %v", observer.disposed)
	}
}

func TestVisitCeasesOnHTTPCease(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {{Status: crawlcapability.FetchCeased}},
	}}
	observer := newObserver()
	visitor := pagevisit.NewPageVisit(
		fetch, pagevisit.AlwaysDue{}, &fakeAbsorption{}, observer, &manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Classification != pagevisit.Ceased {
		t.Fatalf("want ceased, got %v", outcome.Classification)
	}
	if observer.refusals[crawlcapability.RefusalCease] != 1 {
		t.Fatalf("cease not honored: %v", observer.refusals)
	}
}

func TestVisitReportsTransient(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {{Status: crawlcapability.FetchTransient}},
	}}
	visitor := pagevisit.NewPageVisit(
		fetch, pagevisit.AlwaysDue{}, &fakeAbsorption{}, newObserver(), &manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Classification != pagevisit.Transient {
		t.Fatalf("want transient, got %v", outcome.Classification)
	}
}

func TestVisitReportsDeferred(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {{Status: crawlcapability.FetchDeferred, DeferFor: time.Second}},
	}}
	visitor := pagevisit.NewPageVisit(
		fetch, pagevisit.AlwaysDue{}, &fakeAbsorption{}, newObserver(), &manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Classification != pagevisit.Deferred || outcome.DeferFor != time.Second {
		t.Fatalf("want deferred for 1s, got %+v", outcome)
	}
}

func TestVisitFetchErrorFails(t *testing.T) {
	fetch := &fakeFetch{err: errors.New("boom")}
	visitor := pagevisit.NewPageVisit(
		fetch, pagevisit.AlwaysDue{}, &fakeAbsorption{}, newObserver(), &manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err == nil {
		t.Fatal("fetch error should fail the visit")
	}
}

func TestVisitSkipsFetchWhenNotDue(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	absorption := &fakeAbsorption{}
	visitor := pagevisit.NewPageVisit(
		fetch, fakeRecrawl{due: false}, absorption, newObserver(), &manualClock{},
	)

	outcome, err := visitor.Visit(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Classification != pagevisit.NotDue {
		t.Fatalf("want not-due, got %v", outcome.Classification)
	}
}

func TestVisitRecrawlDecisionErrorFails(t *testing.T) {
	visitor := pagevisit.NewPageVisit(
		&fakeFetch{}, fakeRecrawl{err: errors.New("boom")}, &fakeAbsorption{},
		newObserver(), &manualClock{},
	)

	if _, err := visitor.Visit(context.Background(), "http://host/"); err == nil {
		t.Fatal("recrawl decision error should fail the visit")
	}
}
