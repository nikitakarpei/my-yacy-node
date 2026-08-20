package visitdispatch_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/frontier"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitdispatch"
)

type dispatchedVisitor struct {
	mu         sync.Mutex
	visited    []string
	discovered yacycrawlcontract.CanonicalURL
	err        error
}

func (v *dispatchedVisitor) Visit(
	_ context.Context,
	canonicalURL yacycrawlcontract.CanonicalURL,
) (pagevisit.VisitOutcome, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.visited = append(v.visited, canonicalURL.String())
	if v.err != nil {
		return pagevisit.VisitOutcome{}, v.err
	}

	return pagevisit.VisitOutcome{
		Conclusion:     pagevisit.VisitCompleted,
		DiscoveredURLs: []yacycrawlcontract.CanonicalURL{v.discovered},
	}, nil
}

func TestDispatchReportsOnePerVisit(t *testing.T) {
	visitors := visitdispatch.StartVisitors(
		t.Context(),
		&dispatchedVisitor{discovered: canonicalurltest.CanonicalURLOf(t, "http://host/next")},
		2,
	)
	defer visitors.Stop()

	urls := []yacycrawlcontract.CanonicalURL{
		canonicalurltest.CanonicalURLOf(t, "http://host/a"),
		canonicalurltest.CanonicalURLOf(t, "http://host/b"),
		canonicalurltest.CanonicalURLOf(t, "http://host/c"),
	}
	go func() {
		for _, url := range urls {
			visitors.Pending() <- frontier.PendingVisit{URL: url}
		}
	}()

	seen := map[string]int{}
	for range urls {
		result := <-visitors.Completed()
		if result.Err != nil {
			t.Fatalf("visit %s: %v", result.Visit.URL, result.Err)
		}
		if len(result.Outcome.DiscoveredURLs) != 1 {
			t.Fatalf("outcome not carried back: %+v", result.Outcome)
		}
		seen[result.Visit.URL.String()]++
	}
	for _, url := range urls {
		if seen[url.String()] != 1 {
			t.Fatalf("%s reported %d times", url, seen[url.String()])
		}
	}
}

func TestDispatchCarriesVisitError(t *testing.T) {
	failing := &dispatchedVisitor{err: errors.New("boom")}
	visitors := visitdispatch.StartVisitors(t.Context(), failing, 1)
	defer visitors.Stop()

	visitors.Pending() <- frontier.PendingVisit{URL: canonicalurltest.CanonicalURLOf(t, "http://host/")}

	if result := <-visitors.Completed(); result.Err == nil {
		t.Fatal("visit error should reach the caller")
	}
}

func TestStopEndsEveryVisitor(t *testing.T) {
	visitor := &dispatchedVisitor{}
	visitors := visitdispatch.StartVisitors(t.Context(), visitor, 3)

	visitors.Pending() <- frontier.PendingVisit{URL: canonicalurltest.CanonicalURLOf(t, "http://host/")}
	<-visitors.Completed()
	visitors.Stop()

	if len(visitor.visited) != 1 {
		t.Fatalf("visited = %v", visitor.visited)
	}
}
