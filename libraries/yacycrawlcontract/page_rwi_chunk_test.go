package yacycrawlcontract_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPageRWIMetadataChunkRoundTrip(t *testing.T) {
	chunk := yacycrawlcontract.PageRWIMetadataChunk{
		CanonicalURL: "https://example.org/a",
		Metadata: []yacymodel.URLMetadata{
			{
				Address:  "https://example.org/a",
				Title:    "Example",
				Loaded:   yacymodel.Some(yacymodel.NewCalendarDay(2024, time.March, 7)),
				Location: yacymodel.Some(yacymodel.Coordinates{Latitude: 52.52, Longitude: 13.405}),
			},
		},
	}

	data, err := yacycrawlcontract.MarshalPageRWIChunk(chunk)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := yacycrawlcontract.UnmarshalPageRWIChunk(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(yacycrawlcontract.PageRWIChunk(chunk), got) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", chunk, got)
	}
}

func TestPageRWIPostingChunkRoundTrip(t *testing.T) {
	chunk := yacycrawlcontract.PageRWIPostingChunk{
		CanonicalURL: "https://example.org/a",
		Postings: []yacymodel.RWIPosting{
			{
				WordHash: mustHash(t, "wordhash0123"),
				URLHash:  mustURLHash(t, "urlhash01234"),
			},
		},
	}

	data, err := yacycrawlcontract.MarshalPageRWIChunk(chunk)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := yacycrawlcontract.UnmarshalPageRWIChunk(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(yacycrawlcontract.PageRWIChunk(chunk), got) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", chunk, got)
	}
}

func TestUnmarshalPageRWIChunkRejectsUnknownKind(t *testing.T) {
	if _, err := yacycrawlcontract.UnmarshalPageRWIChunk([]byte(`{"Kind":"sonnet"}`)); err == nil {
		t.Fatal("expected error for unknown chunk kind")
	}
}

func TestUnmarshalPageRWIChunkRejectsInvalidJSON(t *testing.T) {
	if _, err := yacycrawlcontract.UnmarshalPageRWIChunk([]byte("not json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
