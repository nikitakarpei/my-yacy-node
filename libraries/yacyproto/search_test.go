package yacyproto_test

import (
	"context"
	"net/url"
	"reflect"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func TestSearchRequestRoundTrip(t *testing.T) {
	t.Parallel()

	req := yacyproto.SearchRequest{
		NetworkName: yacyproto.DefaultNetwork,
		MySeed:      yacymodel.Some(sampleSeed(t, "alpha", "peer-a")),
		Query: []yacymodel.Hash{
			sampleHash(t, "alpha"),
			sampleHash(t, "beta"),
		},
		Exclude: []yacymodel.Hash{
			sampleHash(t, "gamma"),
		},
		URLs: []yacymodel.URLHash{
			sampleURLHash(t, "url-a"),
		},
		Count:            10,
		Time:             3000,
		MaxDist:          5,
		Partitions:       30,
		Abstracts:        yacyproto.SearchAbstractsAuto,
		ContentDom:       yacyproto.ContentDomainText,
		StrictContentDom: true,
		TimezoneOffset:   120,
		Language:         "en",
		Author:           "ada",
		Protocol:         "https",
	}

	got, err := yacyproto.ParseSearchRequest(t.Context(), req.Form())
	if err != nil {
		t.Fatalf("ParseSearchRequest: %v", err)
	}

	if !reflect.DeepEqual(got, req) {
		t.Fatalf("round-trip mismatch:\n got %#v\nwant %#v", got, req)
	}
}

func TestSearchResponseRoundTrip(t *testing.T) {
	t.Parallel()

	alpha := sampleHash(t, "alpha")
	resp := yacyproto.SearchResponse{
		ResponseHeader: yacyproto.ResponseHeader{Version: "1.0", Uptime: 11},
		SearchTime:     120,
		References:     "topic",
		JoinCount:      4,
		Count:          2,
		Resources: []yacyproto.SearchResource{
			sampleSearchResource(t, "url-a"),
			sampleSearchResource(t, "url-b"),
		},
		IndexCount:    map[yacymodel.Hash]int{alpha: 17},
		IndexAbstract: map[yacymodel.Hash]string{alpha: "abc"},
	}

	msg := resp.Encode()
	yacyproto.InjectResponseHeader(msg, resp.Version, resp.Uptime)
	got, err := yacyproto.ParseSearchResponse(context.Background(), msg)
	if err != nil {
		t.Fatalf("ParseSearchResponse: %v", err)
	}

	if !reflect.DeepEqual(got, resp) {
		t.Fatalf("round-trip mismatch:\n got %#v\nwant %#v", got, resp)
	}
}

func TestParseSearchRequestTruncatesRaggedQuery(t *testing.T) {
	t.Parallel()

	full := sampleHash(t, "alpha").String()
	form := url.Values{yacyproto.FieldQuery: {full + "tooshort"}}
	req, err := yacyproto.ParseSearchRequest(t.Context(), form)
	if err != nil {
		t.Fatalf("ParseSearchRequest: %v", err)
	}
	if len(req.Query) != 1 {
		t.Fatalf("Query = %d, want 1 (trailing partial ignored)", len(req.Query))
	}
}

func TestParseSearchRequestTruncatesRaggedExclude(t *testing.T) {
	t.Parallel()

	full := sampleHash(t, "alpha").String()
	form := url.Values{yacyproto.FieldExclude: {full + "tooshort"}}
	req, err := yacyproto.ParseSearchRequest(t.Context(), form)
	if err != nil {
		t.Fatalf("ParseSearchRequest: %v", err)
	}
	if len(req.Exclude) != 1 {
		t.Fatalf("Exclude = %d, want 1 (trailing partial ignored)", len(req.Exclude))
	}
}

func TestParseSearchRequestRejectsUnknownContentDomain(t *testing.T) {
	t.Parallel()

	form := url.Values{yacyproto.FieldContentDom: {"binary"}}
	if _, err := yacyproto.ParseSearchRequest(t.Context(), form); err == nil {
		t.Fatal("expected error for unknown content domain")
	}
}

func TestParseSearchRequestRejectsUnknownStrictContentDom(t *testing.T) {
	t.Parallel()

	form := url.Values{yacyproto.FieldStrictContentDom: {"yes"}}
	if _, err := yacyproto.ParseSearchRequest(t.Context(), form); err == nil {
		t.Fatal("expected error for unknown boolean token")
	}
}

func TestSearchResponseUsesYaCyCountField(t *testing.T) {
	t.Parallel()

	msg := yacyproto.Message{yacyproto.FieldCount: "5"}
	got, err := yacyproto.ParseSearchResponse(context.Background(), msg)
	if err != nil {
		t.Fatalf("ParseSearchResponse: %v", err)
	}

	if got.Count != 5 {
		t.Fatalf("count = %d, want 5", got.Count)
	}
}

func TestParseSearchResponseSkipsMissingAndBadResources(t *testing.T) {
	t.Parallel()

	valid := sampleURLMetadata("url-a")
	msg := yacyproto.Message{
		yacyproto.FieldCount: "3",
		"resource0":          sampleURLMetadataWireForm(t, valid),
		"resource2":          "bad",
	}
	got, err := yacyproto.ParseSearchResponse(context.Background(), msg)
	if err != nil {
		t.Fatalf("ParseSearchResponse: %v", err)
	}
	if len(got.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(got.Resources))
	}
	if !reflect.DeepEqual(got.Resources[0].Metadata, valid) {
		t.Fatalf("resource = %#v, want %#v", got.Resources[0].Metadata, valid)
	}
}

func TestSearchResponsePostingNamesTheDocumentOfItsRow(t *testing.T) {
	t.Parallel()

	resource := sampleSearchResource(t, "url-a")
	msg := yacyproto.SearchResponse{
		Count:     1,
		Resources: []yacyproto.SearchResource{resource},
	}.Encode()

	got, err := yacyproto.ParseSearchResponse(context.Background(), msg)
	if err != nil {
		t.Fatalf("ParseSearchResponse: %v", err)
	}
	if len(got.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(got.Resources))
	}

	documentHash, err := got.Resources[0].Metadata.Hash()
	if err != nil {
		t.Fatalf("url metadata hash: %v", err)
	}
	if got.Resources[0].Posting.URLHash != documentHash {
		t.Fatalf(
			"posting names %q, row names %q",
			got.Resources[0].Posting.URLHash,
			documentHash,
		)
	}
}
