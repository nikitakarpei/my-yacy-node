package yacycrawlcontract

import (
	"reflect"
	"testing"
	"time"
)

func TestPageMarkdownRepresentationRoundTrip(t *testing.T) {
	page := PageMarkdownRepresentation{
		PageReference: PageReference{
			CanonicalURL: "https://example.org/a",
			Title:        "Hi",
			CrawledAt:    time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC),
			Language:     "en",
		},
		Markdown: []byte("# words here"),
	}

	data, err := MarshalPageMarkdownRepresentation(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalPageMarkdownRepresentation(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(page, got) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", page, got)
	}
}

func TestUnmarshalPageMarkdownRepresentationRejectsInvalidJSON(t *testing.T) {
	if _, err := UnmarshalPageMarkdownRepresentation([]byte("not json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
