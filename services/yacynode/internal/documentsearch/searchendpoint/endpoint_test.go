package searchendpoint

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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

func newSearchEndpoint(index searchtest.PostingIndex, documents searchtest.URLDirectory) endpoint {
	return endpoint{
		identity: searchIdentity(),
		results:  searchresult.New(index, documents, 100),
	}
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
	served := newSearchEndpoint(index, searchtest.URLDirectory{Documents: urlMetadata("u1", "u2")})

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
	served := newSearchEndpoint(index, searchtest.URLDirectory{Documents: urlMetadata("u1", "u2")})

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
	served := newSearchEndpoint(index, searchtest.URLDirectory{})

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
	served := newSearchEndpoint(index, documents)

	resp := serveSearch(t, served, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{word},
	})

	if !strings.Contains(resp.References, "kitten") {
		t.Errorf("References = %q, want the title topics", resp.References)
	}
}

func TestEndpointRejectsMalformedCriteria(t *testing.T) {
	served := newSearchEndpoint(searchtest.PostingIndex{}, searchtest.URLDirectory{})

	_, err := served.Serve(context.Background(), yacyproto.SearchRequest{
		NetworkName: "freeworld",
		SiteHash:    "!!",
	})
	if err == nil {
		t.Fatal("Serve accepted a malformed site hash")
	}
}

func TestEndpointSurfacesSearchFailures(t *testing.T) {
	served := endpoint{
		identity: searchIdentity(),
		results: searchresult.New(
			searchtest.FailingPostingIndex{Err: errScanBroken},
			searchtest.URLDirectory{},
			100,
		),
	}

	_, err := served.Serve(context.Background(), yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Query:       []yacymodel.Hash{searchtest.HashFor("w1")},
	})
	if !errors.Is(err, errScanBroken) {
		t.Fatalf("Serve error = %v, want %v", err, errScanBroken)
	}
}

var errScanBroken = errors.New("scan broken")

func TestEndpointRejectsWrongNetwork(t *testing.T) {
	served := newSearchEndpoint(searchtest.PostingIndex{}, searchtest.URLDirectory{})

	resp := serveSearch(t, served, yacyproto.SearchRequest{NetworkName: "othernet"})

	if resp.Count != 0 {
		t.Errorf("Count = %d, want 0 on network mismatch", resp.Count)
	}
}
