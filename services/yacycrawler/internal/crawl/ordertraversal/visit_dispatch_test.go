package ordertraversal

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/frontier"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type dispatchedVisitor struct {
	mu      sync.Mutex
	visited []string
	err     error
}

func (v *dispatchedVisitor) Visit(_ context.Context, url string) (pagevisit.VisitOutcome, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.visited = append(v.visited, url)
	if v.err != nil {
		return pagevisit.VisitOutcome{}, v.err
	}
	return pagevisit.VisitOutcome{
		Conclusion:     pagevisit.VisitCompleted,
		DiscoveredURLs: []string{url + "/next"},
	}, nil
}

func TestDispatchReportsOnePerVisit(t *testing.T) {
	visitors := startVisitors(t.Context(), &dispatchedVisitor{}, 2)
	defer visitors.stop()

	urls := []string{"http://host/a", "http://host/b", "http://host/c"}
	go func() {
		for _, url := range urls {
			visitors.pending <- frontier.PendingVisit{URL: url}
		}
	}()

	seen := map[string]int{}
	for range urls {
		result := <-visitors.results
		if result.err != nil {
			t.Fatalf("visit %s: %v", result.visit.URL, result.err)
		}
		if len(result.outcome.DiscoveredURLs) != 1 {
			t.Fatalf("outcome not carried back: %+v", result.outcome)
		}
		seen[result.visit.URL]++
	}
	for _, url := range urls {
		if seen[url] != 1 {
			t.Fatalf("%s reported %d times", url, seen[url])
		}
	}
}

func TestDispatchCarriesVisitError(t *testing.T) {
	visitors := startVisitors(t.Context(), &dispatchedVisitor{err: errors.New("boom")}, 1)
	defer visitors.stop()

	visitors.pending <- frontier.PendingVisit{URL: "http://host/"}

	if result := <-visitors.results; result.err == nil {
		t.Fatal("visit error should reach the caller")
	}
}

func TestStopEndsEveryVisitor(t *testing.T) {
	visitor := &dispatchedVisitor{}
	visitors := startVisitors(t.Context(), visitor, 3)

	visitors.pending <- frontier.PendingVisit{URL: "http://host/"}
	<-visitors.results
	visitors.stop()

	if len(visitor.visited) != 1 {
		t.Fatalf("visited = %v", visitor.visited)
	}
}
