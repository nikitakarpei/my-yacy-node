package yacycrawlcontract_test

import (
	"reflect"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPageRWIRepresentationFromChunksJoinsMetadataAndPostingsInOrder(t *testing.T) {
	metadata := []yacymodel.URLMetadata{{Title: "Hi"}}
	firstPostings := []yacymodel.RWIPosting{{WordHash: yacymodel.WordHash("fox")}}
	secondPostings := []yacymodel.RWIPosting{
		{WordHash: yacymodel.WordHash("dog")},
		{WordHash: yacymodel.WordHash("cat")},
	}

	got, err := yacycrawlcontract.PageRWIRepresentationFromChunks([]yacycrawlcontract.PageRWIChunk{
		yacycrawlcontract.PageRWIPostingChunk{
			CanonicalURL: "https://example.org/a",
			Postings:     firstPostings,
		},
		yacycrawlcontract.PageRWIMetadataChunk{
			CanonicalURL: "https://example.org/a",
			Metadata:     metadata,
		},
		yacycrawlcontract.PageRWIPostingChunk{
			CanonicalURL: "https://example.org/a",
			Postings:     secondPostings,
		},
	})
	if err != nil {
		t.Fatalf("join: %v", err)
	}

	want := yacycrawlcontract.PageRWIRepresentation{
		CanonicalURL: "https://example.org/a",
		Metadata:     metadata,
		Postings:     append(append([]yacymodel.RWIPosting{}, firstPostings...), secondPostings...),
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("join mismatch:\nwant %#v\ngot  %#v", want, got)
	}
}

func TestPageRWIRepresentationFromChunksRejectsNoMetadataChunk(t *testing.T) {
	_, err := yacycrawlcontract.PageRWIRepresentationFromChunks([]yacycrawlcontract.PageRWIChunk{
		yacycrawlcontract.PageRWIPostingChunk{
			CanonicalURL: "https://example.org/a",
			Postings:     []yacymodel.RWIPosting{{WordHash: yacymodel.WordHash("fox")}},
		},
	})
	if err == nil {
		t.Fatal("expected error for missing metadata chunk")
	}
}

func TestPageRWIRepresentationFromChunksRejectsMoreThanOneMetadataChunk(t *testing.T) {
	_, err := yacycrawlcontract.PageRWIRepresentationFromChunks([]yacycrawlcontract.PageRWIChunk{
		yacycrawlcontract.PageRWIMetadataChunk{CanonicalURL: "https://example.org/a"},
		yacycrawlcontract.PageRWIMetadataChunk{CanonicalURL: "https://example.org/a"},
	})
	if err == nil {
		t.Fatal("expected error for more than one metadata chunk")
	}
}

func TestPageRWIRepresentationFromChunksRejectsDisagreeingCanonicalURL(t *testing.T) {
	_, err := yacycrawlcontract.PageRWIRepresentationFromChunks([]yacycrawlcontract.PageRWIChunk{
		yacycrawlcontract.PageRWIMetadataChunk{CanonicalURL: "https://example.org/a"},
		yacycrawlcontract.PageRWIPostingChunk{
			CanonicalURL: "https://example.org/b",
			Postings:     []yacymodel.RWIPosting{{WordHash: yacymodel.WordHash("fox")}},
		},
	})
	if err == nil {
		t.Fatal("expected error for disagreeing canonical url")
	}
}
