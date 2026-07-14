package pagepublication

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestChunkPageRWIEmitsMetadataFirstThenBoundedPostings(t *testing.T) {
	postings := make([]yacymodel.RWIPosting, postingsPerChunkLimit*2+1)
	for i := range postings {
		postings[i] = yacymodel.RWIPosting{WordHash: yacymodel.WordHash("w")}
	}
	representation := yacycrawlcontract.PageRWIRepresentation{
		CanonicalURL: "https://example.org/a",
		Metadata: []yacymodel.URIMetadataRow{
			{Properties: map[string]string{"u": "urlhash01234"}},
		},
		Postings: postings,
	}

	chunks := chunkPageRWI(representation)

	if len(chunks) != 4 {
		t.Fatalf("chunks = %d, want 4 (1 metadata + 3 posting chunks)", len(chunks))
	}
	metadata, ok := chunks[0].(yacycrawlcontract.PageRWIMetadataChunk)
	if !ok {
		t.Fatalf("first chunk = %T, want PageRWIMetadataChunk", chunks[0])
	}
	if len(metadata.Metadata) != 1 {
		t.Fatalf("metadata chunk rows = %d, want 1", len(metadata.Metadata))
	}
	for i, chunk := range chunks[1:] {
		posting, ok := chunk.(yacycrawlcontract.PageRWIPostingChunk)
		if !ok {
			t.Fatalf("chunk %d = %T, want PageRWIPostingChunk", i, chunk)
		}
		if len(posting.Postings) > postingsPerChunkLimit {
			t.Fatalf("posting chunk %d has %d postings, over limit %d",
				i, len(posting.Postings), postingsPerChunkLimit)
		}
		if posting.CanonicalURL != representation.CanonicalURL {
			t.Fatalf("posting chunk %d url = %q, want %q",
				i, posting.CanonicalURL, representation.CanonicalURL)
		}
	}
}

func TestChunkPageRWIWithoutPostingsEmitsMetadataOnly(t *testing.T) {
	representation := yacycrawlcontract.PageRWIRepresentation{
		CanonicalURL: "https://example.org/a",
		Metadata: []yacymodel.URIMetadataRow{
			{Properties: map[string]string{"u": "urlhash01234"}},
		},
	}

	chunks := chunkPageRWI(representation)

	if len(chunks) != 1 {
		t.Fatalf("chunks = %d, want a single metadata chunk", len(chunks))
	}
	if _, ok := chunks[0].(yacycrawlcontract.PageRWIMetadataChunk); !ok {
		t.Fatalf("chunk = %T, want PageRWIMetadataChunk", chunks[0])
	}
}
