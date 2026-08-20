package redirectrecording_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/redirectrecording"
)

type redirectEdge struct{ requested, canonical string }

type recordingResolutions struct {
	mu       sync.Mutex
	edges    []redirectEdge
	failWith error
}

func (r *recordingResolutions) Record(
	_ context.Context,
	requested, canonical yacycrawlcontract.CanonicalURL,
) error {
	if r.failWith != nil {
		return r.failWith
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.edges = append(r.edges, redirectEdge{
		requested: requested.String(),
		canonical: canonical.String(),
	})
	return nil
}

type recordingFetch struct {
	outcome    pagevisit.FetchOutcome
	err        error
	fetches    int
	gotVersion pagevisit.PageVersion
}

func (f *recordingFetch) Fetch(
	_ context.Context,
	_ string,
	knownVersion pagevisit.PageVersion,
) (pagevisit.FetchOutcome, error) {
	f.fetches++
	f.gotVersion = knownVersion
	if f.err != nil {
		return pagevisit.FetchOutcome{}, f.err
	}
	return f.outcome, nil
}

func fetchedOutcome(finalURL string, chain []string) pagevisit.FetchOutcome {
	return pagevisit.FetchOutcome{
		Status:        pagevisit.FetchSucceeded,
		Page:          fetchedpage.Page{FinalURL: finalURL},
		RedirectChain: chain,
	}
}

func fetch(
	t *testing.T,
	resolutions redirectrecording.RedirectResolutions,
	outcome pagevisit.FetchOutcome,
) (pagevisit.FetchOutcome, *recordingFetch) {
	t.Helper()
	fetcher := &recordingFetch{outcome: outcome}
	got, err := redirectrecording.New(resolutions, fetcher).
		Fetch(context.Background(), "http://host/a", pagevisit.PageVersion{})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	return got, fetcher
}

func TestFetchRecordsEdgePerNonFinalHop(t *testing.T) {
	resolutions := &recordingResolutions{}

	fetch(t, resolutions, fetchedOutcome("http://host/c", []string{
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

func TestFetchRecordsNoEdgeOnDirectFetch(t *testing.T) {
	resolutions := &recordingResolutions{}

	fetch(t, resolutions, fetchedOutcome("http://host/", []string{"http://host/"}))

	if len(resolutions.edges) != 0 {
		t.Fatalf("direct fetch recorded edges: %v", resolutions.edges)
	}
}

func TestFetchRecordsNoEdgeWithoutAFinalURL(t *testing.T) {
	resolutions := &recordingResolutions{}

	fetch(t, resolutions, pagevisit.FetchOutcome{
		Status:        pagevisit.FetchNotModified,
		RedirectChain: []string{"http://host/a"},
	})

	if len(resolutions.edges) != 0 {
		t.Fatalf("a fetch without a final url recorded edges: %v", resolutions.edges)
	}
}

func TestFetchPassesThroughTheOutcome(t *testing.T) {
	outcome, fetcher := fetch(t, &recordingResolutions{}, fetchedOutcome("http://host/", nil))

	if fetcher.fetches != 1 {
		t.Fatalf("wrapped fetcher not reached: %d", fetcher.fetches)
	}
	if outcome.Page.FinalURL != "http://host/" {
		t.Fatalf("outcome = %+v", outcome)
	}
}

func TestFetchPassesThroughTheKnownVersion(t *testing.T) {
	fetcher := &recordingFetch{outcome: fetchedOutcome("http://host/", nil)}
	known := pagevisit.PageVersion{EntityTag: `"etag"`}

	if _, err := redirectrecording.New(&recordingResolutions{}, fetcher).
		Fetch(context.Background(), "http://host/a", known); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	if fetcher.gotVersion != known {
		t.Fatalf("known version = %+v, want %+v", fetcher.gotVersion, known)
	}
}

func TestFetchPropagatesFetchError(t *testing.T) {
	fetcher := &recordingFetch{err: errors.New("boom")}

	if _, err := redirectrecording.New(&recordingResolutions{}, fetcher).
		Fetch(context.Background(), "http://host/a", pagevisit.PageVersion{}); err == nil {
		t.Fatal("a fetch error should reach the caller")
	}
}

func TestFetchSurvivesRecordFailure(t *testing.T) {
	resolutions := &recordingResolutions{failWith: errors.New("bucket down")}

	_, fetcher := fetch(t, resolutions,
		fetchedOutcome("http://host/b", []string{"http://host/a"}))

	if fetcher.fetches != 1 {
		t.Fatal("a failed record should not stop the fetch")
	}
}

func TestFetchSkipsUncanonicalizableURLs(t *testing.T) {
	resolutions := &recordingResolutions{}

	fetch(t, resolutions, fetchedOutcome("::not a url", []string{"http://host/a"}))
	fetch(t, resolutions, fetchedOutcome("http://host/b", []string{"::not a url"}))

	if len(resolutions.edges) != 0 {
		t.Fatalf("uncanonicalizable urls recorded: %v", resolutions.edges)
	}
}
