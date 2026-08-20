package yacycrawlcontract_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/canonicalurltest"
)

func TestPageTextRepresentationRoundTrip(t *testing.T) {
	page := yacycrawlcontract.PageTextRepresentation{
		PageReference: yacycrawlcontract.PageReference{
			CanonicalURL: canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
			Title:        "Hi",
			CrawledAt:    time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC),
			Language:     "en",
		},
		Text: []byte("words here"),
	}

	data, err := yacycrawlcontract.MarshalPageTextRepresentation(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := yacycrawlcontract.UnmarshalPageTextRepresentation(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(page, got) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", page, got)
	}
}

func TestPageTextRepresentationRoundTripsArbitraryTextBytes(t *testing.T) {
	page := yacycrawlcontract.PageTextRepresentation{
		PageReference: yacycrawlcontract.PageReference{
			CanonicalURL: canonicalurltest.CanonicalURLOf(t, "https://example.org/b"),
			CrawledAt:    time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC),
		},
		Text: []byte{0x00, 0x01, 0xff},
	}

	data, err := yacycrawlcontract.MarshalPageTextRepresentation(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := yacycrawlcontract.UnmarshalPageTextRepresentation(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(page, got) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", page, got)
	}
}

func TestUnmarshalPageTextRepresentationRejectsInvalidJSON(t *testing.T) {
	if _, err := yacycrawlcontract.UnmarshalPageTextRepresentation([]byte("not json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
