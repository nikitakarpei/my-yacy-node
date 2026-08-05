package searchendpoint

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type fakeScanner struct {
	postings map[yacymodel.Hash][]yacymodel.RWIPosting
}

func (s fakeScanner) RWICount(context.Context) (int, error) {
	return len(s.postings), nil
}

func (s fakeScanner) ScanWord(
	_ context.Context,
	word yacymodel.Hash,
	visit func(yacymodel.RWIPosting) (bool, error),
) error {
	for _, entry := range s.postings[word] {
		entry.WordHash = word
		keepGoing, err := visit(entry)
		if err != nil {
			return err
		}
		if !keepGoing {
			return nil
		}
	}

	return nil
}

func (s fakeScanner) PostingOf(
	_ context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	for _, entry := range s.postings[word] {
		if entry.URLHash == url {
			entry.WordHash = word

			return entry, true, nil
		}
	}

	return yacymodel.RWIPosting{}, false, nil
}

type fakeDirectory struct {
	documentDirectory map[yacymodel.URLHash]yacymodel.URLMetadata
}

func (d fakeDirectory) MetadataByHash(
	_ context.Context,
	hashes []yacymodel.URLHash,
) ([]yacymodel.URLMetadata, error) {
	out := make([]yacymodel.URLMetadata, 0, len(hashes))
	for _, hash := range hashes {
		if stored, ok := d.documentDirectory[hash]; ok {
			out = append(out, stored)
		}
	}

	return out, nil
}

func (d fakeDirectory) MissingURLs(
	context.Context,
	[]yacymodel.URLHash,
) ([]yacymodel.URLHash, error) {
	return nil, nil
}

func (d fakeDirectory) Count(context.Context) (int, error) {
	return len(d.documentDirectory), nil
}

func hashFor(base string) yacymodel.Hash {
	const filler = "AAAAAAAAAAAA"
	hash, err := yacymodel.ParseHash(base + filler[len(base):])
	if err != nil {
		panic(err)
	}

	return hash
}

func urlHashFor(url string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(hashFor(url).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func postingEntry(word yacymodel.Hash, url string) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash: word,
		URLHash:  urlHashFor(url),
		Hits:     1,
	}
}

func urlMetadata(ids ...string) map[yacymodel.URLHash]yacymodel.URLMetadata {
	metadata := make(map[yacymodel.URLHash]yacymodel.URLMetadata, len(ids))
	for _, id := range ids {
		metadata[urlHashFor(id)] = yacymodel.URLMetadata{Address: "http://example.com/" + id}
	}

	return metadata
}

func searchIdentity() nodeidentity.Identity {
	return nodeidentity.Identity{Hash: yacymodel.WordHash("self"), NetworkName: "freeworld"}
}

func newSearchEndpoint(index fakeScanner, documents fakeDirectory) endpoint {
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
	word := hashFor("w1")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1"), postingEntry(word, "u2")},
	}}
	served := newSearchEndpoint(index, fakeDirectory{documentDirectory: urlMetadata("u1", "u2")})

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
	word1, word2 := hashFor("w1"), hashFor("w2")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {postingEntry(word1, "u1"), postingEntry(word1, "u2")},
		word2: {postingEntry(word2, "u2")},
	}}
	served := newSearchEndpoint(index, fakeDirectory{documentDirectory: urlMetadata("u1", "u2")})

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
	word := hashFor("w1")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1"), postingEntry(word, "u2")},
	}}
	served := newSearchEndpoint(index, fakeDirectory{})

	resp := serveSearch(t, served, yacyproto.SearchRequest{
		NetworkName: "freeworld",
		Abstracts:   yacyproto.SearchAbstracts(word.String()),
	})

	if resp.IndexCount[word] != 2 {
		t.Errorf("IndexCount = %v, want 2 for term", resp.IndexCount)
	}
}

func TestEndpointRejectsWrongNetwork(t *testing.T) {
	served := newSearchEndpoint(fakeScanner{}, fakeDirectory{})

	resp := serveSearch(t, served, yacyproto.SearchRequest{NetworkName: "othernet"})

	if resp.Count != 0 {
		t.Errorf("Count = %d, want 0 on network mismatch", resp.Count)
	}
}
