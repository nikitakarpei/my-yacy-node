package redirectrecording_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/redirectrecording"
)

type redirectEdge struct{ requested, canonical string }

type recordingResolutions struct {
	mu       sync.Mutex
	edges    []redirectEdge
	failWith error
}

func (r *recordingResolutions) Record(_ context.Context, requested, canonical string) error {
	if r.failWith != nil {
		return r.failWith
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.edges = append(r.edges, redirectEdge{requested: requested, canonical: canonical})
	return nil
}

type recordingAbsorption struct {
	links    []string
	absorbed int
}

func (a *recordingAbsorption) Absorb(
	context.Context,
	fetchedpage.Page,
) ([]string, error) {
	a.absorbed++
	return a.links, nil
}

func fetched(finalURL string, chain []string) fetchedpage.Page {
	return fetchedpage.Page{
		FinalURL:      finalURL,
		RedirectChain: chain,
	}
}

func absorb(
	t *testing.T,
	resolutions redirectrecording.RedirectResolutions,
	page fetchedpage.Page,
) ([]string, *recordingAbsorption) {
	t.Helper()
	absorption := &recordingAbsorption{links: []string{"http://host/next"}}
	links, err := redirectrecording.New(resolutions, absorption).
		Absorb(context.Background(), page)
	if err != nil {
		t.Fatalf("absorb: %v", err)
	}
	return links, absorption
}

func TestAbsorbRecordsEdgePerNonFinalHop(t *testing.T) {
	resolutions := &recordingResolutions{}

	absorb(t, resolutions, fetched("http://host/c", []string{
		"http://host/a", "http://host/b", "http://host/c",
	}))

	want := []redirectEdge{
		{requested: "http://host/a", canonical: "http://host/c"},
		{requested: "http://host/b", canonical: "http://host/c"},
	}
	if len(resolutions.edges) != len(want) {
		t.Fatalf("edges = %v, want %v", resolutions.edges, want)
	}
	for i := range want {
		if resolutions.edges[i] != want[i] {
			t.Fatalf("edge[%d] = %v, want %v", i, resolutions.edges[i], want[i])
		}
	}
}

func TestAbsorbRecordsNoEdgeOnDirectFetch(t *testing.T) {
	resolutions := &recordingResolutions{}

	absorb(t, resolutions, fetched("http://host/", []string{"http://host/"}))

	if len(resolutions.edges) != 0 {
		t.Fatalf("direct fetch recorded edges: %v", resolutions.edges)
	}
}

func TestAbsorbPassesThroughDiscoveredLinks(t *testing.T) {
	links, absorption := absorb(t, &recordingResolutions{}, fetched("http://host/", nil))

	if absorption.absorbed != 1 {
		t.Fatalf("absorption not reached: %d", absorption.absorbed)
	}
	if len(links) != 1 || links[0] != "http://host/next" {
		t.Fatalf("links = %v", links)
	}
}

func TestAbsorbSurvivesRecordFailure(t *testing.T) {
	resolutions := &recordingResolutions{failWith: errors.New("bucket down")}

	_, absorption := absorb(t, resolutions, fetched("http://host/b", []string{"http://host/a"}))

	if absorption.absorbed != 1 {
		t.Fatal("a failed record should not stop absorption")
	}
}

func TestAbsorbSkipsUncanonicalizableURLs(t *testing.T) {
	resolutions := &recordingResolutions{}

	absorb(t, resolutions, fetched("::not a url", []string{"http://host/a"}))
	absorb(t, resolutions, fetched("http://host/b", []string{"::not a url"}))

	if len(resolutions.edges) != 0 {
		t.Fatalf("uncanonicalizable urls recorded: %v", resolutions.edges)
	}
}
