package searchendpoint

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func postingEntry(word yacymodel.Hash, url string) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash: word,
		URLHash:  searchtest.URLHashFor(url),
		Hits:     1,
	}
}

func urlMetadata(ids ...string) map[yacymodel.URLHash]yacymodel.URLMetadata {
	metadata := make(map[yacymodel.URLHash]yacymodel.URLMetadata, len(ids))
	for _, id := range ids {
		metadata[searchtest.URLHashFor(id)] = yacymodel.URLMetadata{
			Address: "http://example.com/" + id,
		}
	}

	return metadata
}

func searchIdentity() nodeidentity.Identity {
	return nodeidentity.Identity{Hash: yacymodel.WordHash("self"), NetworkName: "freeworld"}
}

func newSearchEndpoint(
	t *testing.T,
	index searchtest.PostingIndex,
	documents searchtest.URLDirectory,
) endpoint {
	t.Helper()

	served, _ := observedEndpoint(t, searchresult.New(index, documents, 100))

	return served
}

func observedEndpoint(
	t *testing.T,
	results searchresult.Results,
) (endpoint, *prometheus.Registry) {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatal(err)
	}
	registry := prometheus.NewRegistry()

	return endpoint{
		identity: searchIdentity(),
		results:  results,
		observation: searchObservation{
			metrics:      searchmetrics.NewSearchMetrics(registry),
			nodePosition: yacymodel.DHTRingPositionOf(searchIdentity().Hash),
			partitions:   partitions,
		},
	}, registry
}

func serveSearch(
	t *testing.T,
	served endpoint,
	req yacyproto.SearchRequest,
) yacyproto.SearchResponse {
	t.Helper()

	resp, err := served.Serve(context.Background(), req)
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}

	return resp
}

func TestEndpointJoinsAndAnswers(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1"), postingEntry(word, "u2")},
	}}
	served := newSearchEndpoint(
		t,
		index,
		searchtest.URLDirectory{Documents: urlMetadata("u1", "u2")},
	)

	resp := serveSearch(t, served, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{word},
		Count:       10,
	})

	if resp.Count != 2 || resp.JoinCount != 2 {
		t.Errorf("Count = %d, JoinCount = %d, want 2/2", resp.Count, resp.JoinCount)
	}
}

func TestEndpointReportsTermWithMostMatches(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {postingEntry(word1, "u1"), postingEntry(word1, "u2")},
		word2: {postingEntry(word2, "u2")},
	}}
	served := newSearchEndpoint(
		t,
		index,
		searchtest.URLDirectory{Documents: urlMetadata("u1", "u2")},
	)

	resp := serveSearch(t, served, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{word1, word2},
		Abstracts:   yacyproto.SearchAbstractsAuto,
	})

	if resp.Count != 1 {
		t.Errorf("Count = %d, want 1", resp.Count)
	}
	if len(resp.IndexAbstract) == 0 {
		t.Error("IndexAbstract empty, want reported term")
	}
}

func TestEndpointReportsRequestedTerms(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1"), postingEntry(word, "u2")},
	}}
	served := newSearchEndpoint(t, index, searchtest.URLDirectory{})

	resp := serveSearch(t, served, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Abstracts:   yacyproto.SearchAbstracts(word.String()),
	})

	if resp.IndexCount[word] != 2 {
		t.Errorf("IndexCount = %v, want 2 for term", resp.IndexCount)
	}
}

func TestEndpointAnswersWithTitleTopics(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1")},
	}}
	documents := searchtest.URLDirectory{Documents: map[yacymodel.URLHash]yacymodel.URLMetadata{
		searchtest.URLHashFor("u1"): {
			Address: "http://example.com/u1",
			Title:   "orange kitten pictures",
		},
	}}
	served := newSearchEndpoint(t, index, documents)

	resp := serveSearch(t, served, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{word},
	})

	if !strings.Contains(resp.References, "kitten") {
		t.Errorf("References = %q, want the title topics", resp.References)
	}
}

func TestEndpointRejectsMalformedCriteria(t *testing.T) {
	served := newSearchEndpoint(t, searchtest.PostingIndex{}, searchtest.URLDirectory{})

	_, err := served.Serve(context.Background(), yacyproto.SearchRequest{
		NetworkName: "freeworld",
		SiteHash:    "!!",
	})
	if err == nil {
		t.Fatal("Serve accepted a malformed site hash")
	}
}

func TestEndpointSurfacesSearchFailures(t *testing.T) {
	served, registry := observedEndpoint(t, searchresult.New(
		searchtest.FailingPostingIndex{Err: errScanBroken},
		searchtest.URLDirectory{},
		100,
	))

	_, err := served.Serve(context.Background(), yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{searchtest.HashFor("w1")},
	})
	if !errors.Is(err, errScanBroken) {
		t.Fatalf("Serve error = %v, want %v", err, errScanBroken)
	}
	if got := searchesEnded(t, registry, searchmetrics.SearchIndexFailure); got != 1 {
		t.Errorf("index_failure searches = %v, want 1", got)
	}
}

var errScanBroken = errors.New("scan broken")

func TestEndpointRejectsWrongNetwork(t *testing.T) {
	served := newSearchEndpoint(t, searchtest.PostingIndex{}, searchtest.URLDirectory{})

	resp := serveSearch(t, served, yacyproto.SearchRequest{NetworkName: "othernet"})

	if resp.Count != 0 {
		t.Errorf("Count = %d, want 0 on network mismatch", resp.Count)
	}
}

func TestEndpointObservesServedOutcomesAndTermPresence(t *testing.T) {
	word, missingWord := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1")},
	}}
	served, registry := observedEndpoint(t, searchresult.New(
		index,
		searchtest.URLDirectory{Documents: urlMetadata("u1")},
		100,
	))

	serveSearch(t, served, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{word},
		Count:       10,
	})
	serveSearch(t, served, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{missingWord},
		Count:       10,
	})

	if got := searchesEnded(t, registry, searchmetrics.SearchServedWithResults); got != 1 {
		t.Errorf("served_with_results searches = %v, want 1", got)
	}
	if got := searchesEnded(t, registry, searchmetrics.SearchServedNoResults); got != 1 {
		t.Errorf("served_no_results searches = %v, want 1", got)
	}
	if got := termsObserved(t, registry, "in_index"); got != 1 {
		t.Errorf("in_index terms = %v, want 1", got)
	}
	if got := termsObserved(t, registry, "not_in_index"); got != 1 {
		t.Errorf("not_in_index terms = %v, want 1", got)
	}
}

func TestEndpointObservesNetworkMismatch(t *testing.T) {
	served, registry := observedEndpoint(
		t,
		searchresult.New(searchtest.PostingIndex{}, searchtest.URLDirectory{}, 100),
	)

	serveSearch(t, served, yacyproto.SearchRequest{NetworkName: "othernet"})

	if got := searchesEnded(t, registry, searchmetrics.SearchNetworkMismatch); got != 1 {
		t.Errorf("network_mismatch searches = %v, want 1", got)
	}
}

func TestEndpointObservesInvalidCriteria(t *testing.T) {
	served, registry := observedEndpoint(
		t,
		searchresult.New(searchtest.PostingIndex{}, searchtest.URLDirectory{}, 100),
	)

	_, err := served.Serve(context.Background(), yacyproto.SearchRequest{
		NetworkName: "freeworld",
		SiteHash:    "!!",
	})
	if err == nil {
		t.Fatal("Serve accepted a malformed site hash")
	}
	if got := searchesEnded(t, registry, searchmetrics.SearchInvalidCriteria); got != 1 {
		t.Errorf("invalid_criteria searches = %v, want 1", got)
	}
}

func TestEndpointObservesDeadlineAndMetadataFailures(t *testing.T) {
	deadlineServed, deadlineRegistry := observedEndpoint(t, searchresult.New(
		searchtest.FailingPostingIndex{Err: context.DeadlineExceeded},
		searchtest.URLDirectory{},
		100,
	))
	if _, err := deadlineServed.Serve(context.Background(), yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{searchtest.HashFor("w1")},
	}); err == nil {
		t.Fatal("Serve ignored an exceeded deadline")
	}
	if got := searchesEnded(t, deadlineRegistry, searchmetrics.SearchDeadlineExceeded); got != 1 {
		t.Errorf("deadline_exceeded searches = %v, want 1", got)
	}

	metadataServed, metadataRegistry := observedEndpoint(t, searchresult.New(
		searchtest.PostingIndex{},
		searchtest.FailingURLDirectory{Err: errScanBroken},
		100,
	))
	if _, err := metadataServed.Serve(context.Background(), yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{searchtest.HashFor("w1")},
	}); err == nil {
		t.Fatal("Serve ignored a failing document directory")
	}
	if got := searchesEnded(t, metadataRegistry, searchmetrics.SearchMetadataFailure); got != 1 {
		t.Errorf("metadata_failure searches = %v, want 1", got)
	}
}

func TestEndpointObservesUnsupportedOptions(t *testing.T) {
	served, registry := observedEndpoint(
		t,
		searchresult.New(searchtest.PostingIndex{}, searchtest.URLDirectory{}, 100),
	)

	serveSearch(t, served, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Prefer:      "www",
	})

	families := gatheredFamilies(t, registry)
	value, found := labeledCounter(
		families,
		"documentsearch_unsupported_options_requested_total",
		"prefer",
	)
	if !found || value != 1 {
		t.Errorf("prefer option requests = %v (found %v), want 1", value, found)
	}
}

func searchesEnded(
	t *testing.T,
	registry *prometheus.Registry,
	outcome searchmetrics.SearchOutcome,
) float64 {
	t.Helper()

	value, _ := labeledCounter(
		gatheredFamilies(t, registry),
		"documentsearch_searches_total",
		string(outcome),
	)

	return value
}

func termsObserved(t *testing.T, registry *prometheus.Registry, presence string) uint64 {
	t.Helper()

	for _, family := range gatheredFamilies(t, registry) {
		if family.GetName() != "documentsearch_query_term_ring_fraction" {
			continue
		}
		for _, metric := range family.GetMetric() {
			if labelValueOf(metric) == presence {
				return metric.GetHistogram().GetSampleCount()
			}
		}
	}

	return 0
}

func gatheredFamilies(t *testing.T, registry *prometheus.Registry) []*dto.MetricFamily {
	t.Helper()

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	return families
}

func labeledCounter(
	families []*dto.MetricFamily,
	name string,
	labelValue string,
) (float64, bool) {
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			if labelValueOf(metric) == labelValue {
				return metric.GetCounter().GetValue(), true
			}
		}
	}

	return 0, false
}

func labelValueOf(metric *dto.Metric) string {
	if len(metric.GetLabel()) == 0 {
		return ""
	}

	return metric.GetLabel()[0].GetValue()
}
