package searchendpoint_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func TestEndpointJoinsAndAnswers(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1"), postingEntry(word, "u2")},
	}}
	mux := mountedSearch(t, index, searchtest.URLDirectory{Documents: urlMetadata("u1", "u2")})

	resp := search(t, mux, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{word},
		Count:       10,
	})

	if resp.Count != 2 || resp.JoinCount != 2 {
		t.Errorf("Count = %d, JoinCount = %d, want 2/2", resp.Count, resp.JoinCount)
	}
}

func TestEndpointAnswersWithThePostingThatMatchedEachDocument(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1")},
	}}
	mux := mountedSearch(t, index, searchtest.URLDirectory{Documents: urlMetadata("u1")})

	resp := search(t, mux, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{word},
		Count:       10,
	})

	if len(resp.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resp.Resources))
	}
	posting, ok := resp.Resources[0].Posting.Get()
	if !ok {
		t.Fatal("resource carries no posting")
	}
	if posting.URLHash != documentHashOf("u1") {
		t.Errorf("posting names %q, want %q", posting.URLHash, documentHashOf("u1"))
	}
	if posting.Hits != 1 {
		t.Errorf("posting hits = %d, want 1", posting.Hits)
	}
}

func TestEndpointReportsTermWithMostMatches(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {postingEntry(word1, "u1"), postingEntry(word1, "u2")},
		word2: {postingEntry(word2, "u2")},
	}}
	mux := mountedSearch(t, index, searchtest.URLDirectory{Documents: urlMetadata("u1", "u2")})

	resp := search(t, mux, yacyproto.SearchRequest{
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
	mux := mountedSearch(t, index, searchtest.URLDirectory{})

	resp := search(t, mux, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Abstracts:   yacyproto.SearchAbstracts(word.String()),
	})

	if len(resp.IndexAbstract[word]) == 0 {
		t.Errorf("IndexAbstract = %v, want the requested term", resp.IndexAbstract)
	}
	if len(resp.IndexCount) != 0 {
		t.Errorf("IndexCount = %v, want none without a query", resp.IndexCount)
	}
}

func TestEndpointAnswersWithTitleTopics(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1")},
	}}
	documents := searchtest.URLDirectory{Documents: map[yacymodel.URLHash]yacymodel.URLMetadata{
		documentHashOf("u1"): {
			Address: documentAddressOf("u1"),
			Title:   "orange kitten pictures",
		},
	}}
	mux := mountedSearch(t, index, documents)

	resp := search(t, mux, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{word},
	})

	if !strings.Contains(resp.References, "kitten") {
		t.Errorf("References = %q, want the title topics", resp.References)
	}
}

func TestEndpointRejectsMalformedCriteria(t *testing.T) {
	mux := mountedSearch(t, searchtest.PostingIndex{}, searchtest.URLDirectory{})

	rec := postSearch(t, mux, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		SiteHash:    "!!",
	})
	if rec.Code == 200 {
		t.Fatal("search accepted a malformed site hash")
	}
}

func TestEndpointSurfacesSearchFailures(t *testing.T) {
	mux, registry := mountedSearchResults(t, searchresult.New(
		openVault(t),
		termpostings.New(searchtest.FailingPostingIndex{Err: errScanBroken}, 100),
		searchtest.URLDirectory{},
	))

	rec := postSearch(t, mux, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{searchtest.HashFor("w1")},
	})
	if rec.Code == 200 {
		t.Fatal("search succeeded despite a broken index")
	}
	if got := searchesEnded(t, registry, searchmetrics.SearchIndexFailure); got != 1 {
		t.Errorf("index_failure searches = %v, want 1", got)
	}
}

var errScanBroken = errors.New("scan broken")

func TestEndpointRejectsWrongNetwork(t *testing.T) {
	mux := mountedSearch(t, searchtest.PostingIndex{}, searchtest.URLDirectory{})

	resp := search(t, mux, yacyproto.SearchRequest{NetworkName: "othernet"})

	if resp.Count != 0 {
		t.Errorf("Count = %d, want 0 on network mismatch", resp.Count)
	}
}

func TestEndpointObservesServedOutcomesAndTermPresence(t *testing.T) {
	word, missingWord := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1")},
	}}
	mux, registry := mountedSearchResults(t, searchresult.New(
		openVault(t),
		termpostings.New(index, 100),
		searchtest.URLDirectory{Documents: urlMetadata("u1")},
	))

	search(t, mux, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{word},
		Count:       10,
	})
	search(t, mux, yacyproto.SearchRequest{
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
	mux, registry := mountedSearchResults(
		t,
		searchresult.New(
			openVault(t),
			termpostings.New(searchtest.PostingIndex{}, 100),
			searchtest.URLDirectory{},
		),
	)

	search(t, mux, yacyproto.SearchRequest{NetworkName: "othernet"})

	if got := searchesEnded(t, registry, searchmetrics.SearchNetworkMismatch); got != 1 {
		t.Errorf("network_mismatch searches = %v, want 1", got)
	}
}

func TestEndpointObservesInvalidCriteria(t *testing.T) {
	mux, registry := mountedSearchResults(
		t,
		searchresult.New(
			openVault(t),
			termpostings.New(searchtest.PostingIndex{}, 100),
			searchtest.URLDirectory{},
		),
	)

	rec := postSearch(t, mux, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		SiteHash:    "!!",
	})
	if rec.Code == 200 {
		t.Fatal("search accepted a malformed site hash")
	}
	if got := searchesEnded(t, registry, searchmetrics.SearchInvalidCriteria); got != 1 {
		t.Errorf("invalid_criteria searches = %v, want 1", got)
	}
}

func TestEndpointObservesDeadlineAndMetadataFailures(t *testing.T) {
	deadlineMux, deadlineRegistry := mountedSearchResults(t, searchresult.New(
		openVault(t),
		termpostings.New(searchtest.FailingPostingIndex{Err: context.DeadlineExceeded}, 100),
		searchtest.URLDirectory{},
	))
	if rec := postSearch(t, deadlineMux, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{searchtest.HashFor("w1")},
	}); rec.Code == 200 {
		t.Fatal("search ignored an exceeded deadline")
	}
	if got := searchesEnded(t, deadlineRegistry, searchmetrics.SearchDeadlineExceeded); got != 1 {
		t.Errorf("deadline_exceeded searches = %v, want 1", got)
	}

	metadataMux, metadataRegistry := mountedSearchResults(t, searchresult.New(
		openVault(t),
		termpostings.New(searchtest.PostingIndex{}, 100),
		searchtest.FailingURLDirectory{Err: errScanBroken},
	))
	if rec := postSearch(t, metadataMux, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{searchtest.HashFor("w1")},
	}); rec.Code == 200 {
		t.Fatal("search ignored a failing document directory")
	}
	if got := searchesEnded(t, metadataRegistry, searchmetrics.SearchMetadataFailure); got != 1 {
		t.Errorf("metadata_failure searches = %v, want 1", got)
	}
}

func TestEndpointObservesUnsupportedOptions(t *testing.T) {
	mux, registry := mountedSearchResults(
		t,
		searchresult.New(
			openVault(t),
			termpostings.New(searchtest.PostingIndex{}, 100),
			searchtest.URLDirectory{},
		),
	)

	search(t, mux, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Prefer:      "www",
	})

	families := gatheredFamilies(t, registry)
	value, found := labeledCounter(
		families,
		"yacynode_documentsearch_unsupported_options_requested_total",
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
		"yacynode_documentsearch_searches_total",
		string(outcome),
	)

	return value
}

func termsObserved(t *testing.T, registry *prometheus.Registry, presence string) uint64 {
	t.Helper()

	for _, family := range gatheredFamilies(t, registry) {
		if family.GetName() != "yacynode_documentsearch_query_term_ring_fraction" {
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

func openVault(t *testing.T) *vault.Vault {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = v.Close() })

	return v
}
