package yacycrawlcontract

import (
	"reflect"
	"testing"
	"time"
)

func TestPageTextRepresentationRoundTrip(t *testing.T) {
	page := PageTextRepresentation{
		PageReference: PageReference{
			CanonicalURL: "https://example.org/a",
			Title:        "Hi",
			CrawledAt:    time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC),
			Language:     "en",
		},
		Text: []byte("words here"),
	}

	data, err := MarshalPageTextRepresentation(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalPageTextRepresentation(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(page, got) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", page, got)
	}
}

func TestPageTextRepresentationRoundTripsArbitraryTextBytes(t *testing.T) {
	page := PageTextRepresentation{
		PageReference: PageReference{
			CanonicalURL: "https://example.org/b",
			CrawledAt:    time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC),
		},
		Text: []byte{0x00, 0x01, 0xff},
	}

	data, err := MarshalPageTextRepresentation(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalPageTextRepresentation(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(page, got) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", page, got)
	}
}

func TestUnmarshalPageTextRepresentationRejectsInvalidJSON(t *testing.T) {
	if _, err := UnmarshalPageTextRepresentation([]byte("not json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
