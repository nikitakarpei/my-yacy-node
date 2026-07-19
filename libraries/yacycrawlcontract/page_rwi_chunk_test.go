package yacycrawlcontract

import (
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPageRWIMetadataChunkRoundTrip(t *testing.T) {
	chunk := PageRWIMetadataChunk{
		CanonicalURL: "https://example.org/a",
		Metadata: []yacymodel.URLMetadata{
			{
				Address:  "https://example.org/a",
				Title:    "Example",
				Loaded:   yacymodel.NewCalendarDay(2024, time.March, 7),
				Location: yacymodel.Coordinates{Latitude: 52.52, Longitude: 13.405},
			},
		},
	}

	data, err := MarshalPageRWIChunk(chunk)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalPageRWIChunk(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(PageRWIChunk(chunk), got) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", chunk, got)
	}
}

func TestPageRWIPostingChunkRoundTrip(t *testing.T) {
	chunk := PageRWIPostingChunk{
		CanonicalURL: "https://example.org/a",
		Postings: []yacymodel.RWIPosting{
			{
				WordHash: yacymodel.Hash("wordhash0123"),
				URLHash:  yacymodel.URLHash("urlhash01234"),
			},
		},
	}

	data, err := MarshalPageRWIChunk(chunk)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalPageRWIChunk(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(PageRWIChunk(chunk), got) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", chunk, got)
	}
}

func TestUnmarshalPageRWIChunkRejectsUnknownKind(t *testing.T) {
	if _, err := UnmarshalPageRWIChunk([]byte(`{"Kind":"sonnet"}`)); err == nil {
		t.Fatal("expected error for unknown chunk kind")
	}
}

func TestUnmarshalPageRWIChunkRejectsInvalidJSON(t *testing.T) {
	if _, err := UnmarshalPageRWIChunk([]byte("not json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
