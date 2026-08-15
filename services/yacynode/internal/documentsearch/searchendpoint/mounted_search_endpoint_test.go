package searchendpoint_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchendpoint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	searchNetwork      = "freeworld"
	maxPostingsPerTerm = 100
)

type searchRuntimeStatus struct{}

func (searchRuntimeStatus) Version(context.Context) string { return "1.0" }

func (searchRuntimeStatus) Uptime(context.Context) int { return 0 }

func searchIdentity() nodeidentity.Identity {
	return nodeidentity.Identity{Hash: yacymodel.WordHash("self"), NetworkName: searchNetwork}
}

func mountedSearch(
	t *testing.T,
	index searchtest.PostingIndex,
	documents searchtest.URLDirectory,
) *http.ServeMux {
	t.Helper()

	mux, _ := mountedSearchResults(t, searchresult.New(index, documents, maxPostingsPerTerm))

	return mux
}

func mountedSearchResults(
	t *testing.T,
	results searchresult.Results,
) (*http.ServeMux, *prometheus.Registry) {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatal(err)
	}
	registry := prometheus.NewRegistry()
	mux := http.NewServeMux()
	router := httpguard.NewWireRouter(mux, httpguard.WireGate{
		Guard: httpguard.NewRequestGuard(
			httpguard.DefaultMaxBodyBytes,
			httpguard.DefaultRequestTimeout,
		),
		Respond: httpguard.NewWireResponder(searchRuntimeStatus{}),
		Address: httpguard.NewClientAddressResolver(nil),
	})
	searchendpoint.Mount(
		router,
		searchIdentity(),
		results,
		searchmetrics.NewSearchMetrics(registry),
		partitions,
	)

	return mux, registry
}

func postSearch(
	t *testing.T,
	mux *http.ServeMux,
	req yacyproto.SearchRequest,
) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		yacyproto.PathSearch,
		strings.NewReader(req.Form().Encode()),
	)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	mux.ServeHTTP(rec, httpReq)

	return rec
}

func search(
	t *testing.T,
	mux *http.ServeMux,
	req yacyproto.SearchRequest,
) yacyproto.SearchResponse {
	t.Helper()

	rec := postSearch(t, mux, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %q", rec.Code, rec.Body.String())
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	resp, err := yacyproto.ParseSearchResponse(
		context.Background(),
		yacyproto.ParseMessage(string(body)),
	)
	if err != nil {
		t.Fatalf("ParseSearchResponse: %v", err)
	}

	return resp
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

func postingEntry(word yacymodel.Hash, url string) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash: word,
		URLHash:  searchtest.URLHashFor(url),
		Hits:     1,
	}
}
