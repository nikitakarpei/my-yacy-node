package yacycrawlcontract

import (
	"reflect"
	"testing"
	"time"
)

func TestPageContentRepresentationRoundTrip(t *testing.T) {
	page := PageContentRepresentation{
		CanonicalURL: "https://example.org/a",
		Title:        "Hi",
		CrawledAt:    time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC),
		Language:     "en",
		Body:         []byte("words here"),
	}

	data, err := MarshalPageContentRepresentation(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalPageContentRepresentation(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(page, got) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", page, got)
	}
}

func TestPageContentRepresentationRoundTripsArbitraryBodyBytes(t *testing.T) {
	page := PageContentRepresentation{
		CanonicalURL: "https://example.org/b",
		CrawledAt:    time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC),
		Body:         []byte{0x00, 0x01, 0xff},
	}

	data, err := MarshalPageContentRepresentation(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalPageContentRepresentation(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(page, got) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", page, got)
	}
}

func TestUnmarshalPageContentRepresentationRejectsInvalidJSON(t *testing.T) {
	if _, err := UnmarshalPageContentRepresentation([]byte("not json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
