package yacyproto_test

import (
	"context"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func TestTransferURLRequestRoundTrip(t *testing.T) {
	t.Parallel()

	req := yacyproto.TransferURLRequest{
		NetworkName: yacyproto.DefaultNetwork,
		Iam:         sampleHash(t, "alpha"),
		YouAre:      sampleHash(t, "beta"),
		URLCount:    2,
		URLs: []yacymodel.URLMetadata{
			sampleURLMetadata(t, "url-a"),
			sampleURLMetadata(t, "url-b"),
		},
	}

	got, err := yacyproto.ParseTransferURLRequest(context.Background(), req.Form())
	if err != nil {
		t.Fatalf("ParseTransferURLRequest: %v", err)
	}

	if !reflect.DeepEqual(got, req) {
		t.Fatalf("round-trip mismatch:\n got %#v\nwant %#v", got, req)
	}
}

func TestTransferURLResponseRoundTrip(t *testing.T) {
	t.Parallel()

	resp := yacyproto.TransferURLResponse{
		ResponseHeader: yacyproto.ResponseHeader{Version: "1.0", Uptime: 9},
		Result:         yacyproto.ResultErrorNotGranted,
		Double:         3,
		ErrorURL: []yacymodel.URLHash{
			sampleURLHash(t, "url-a"),
		},
	}

	fields := resp.Encode()
	yacyproto.InjectResponseHeader(fields, resp.Version, resp.Uptime)
	got, err := yacyproto.ParseTransferURLResponse(fields)
	if err != nil {
		t.Fatalf("ParseTransferURLResponse: %v", err)
	}

	if !reflect.DeepEqual(got, resp) {
		t.Fatalf("round-trip mismatch:\n got %#v\nwant %#v", got, resp)
	}
}

func TestParseTransferURLRequestRejectsBadYouAre(t *testing.T) {
	t.Parallel()

	form := url.Values{yacyproto.FieldYouAre: {"!!"}}
	if _, err := yacyproto.ParseTransferURLRequest(context.Background(), form); err == nil {
		t.Fatal("expected error for malformed youare hash")
	}
}

func TestParseTransferURLRequestSkipsMissingDeclaredURL(t *testing.T) {
	t.Parallel()

	form := url.Values{
		yacyproto.FieldIam:      {sampleHash(t, "alpha").String()},
		yacyproto.FieldYouAre:   {sampleHash(t, "beta").String()},
		yacyproto.FieldURLCount: {"2"},
		"url0":                  {sampleURLMetadataWireForm(t, sampleURLMetadata(t, "url-a"))},
	}
	req, err := yacyproto.ParseTransferURLRequest(context.Background(), form)
	if err != nil {
		t.Fatalf("ParseTransferURLRequest: %v", err)
	}
	if len(req.URLs) != 1 {
		t.Fatalf("URLs = %d, want 1 (missing url1 skipped)", len(req.URLs))
	}
}

func TestParseTransferURLRequestReadsAtMostTheRowsOfOneTransfer(t *testing.T) {
	t.Parallel()

	const maxRWIPostingsPerTransfer = 1000
	rowCount := maxRWIPostingsPerTransfer + 1
	form := url.Values{yacyproto.FieldURLCount: {strconv.Itoa(rowCount)}}
	row := sampleURLMetadataWireForm(t, sampleURLMetadata(t, "url-a"))
	for i := range rowCount {
		form.Set("url"+strconv.Itoa(i), row)
	}

	req, err := yacyproto.ParseTransferURLRequest(t.Context(), form)
	if err != nil {
		t.Fatalf("ParseTransferURLRequest: %v", err)
	}
	if len(req.URLs) != maxRWIPostingsPerTransfer {
		t.Fatalf("URLs = %d, want %d", len(req.URLs), maxRWIPostingsPerTransfer)
	}
}

func TestParseTransferURLRequestEndsAHugeDeclaredCountAtTheRowLimit(t *testing.T) {
	t.Parallel()

	form := url.Values{
		yacyproto.FieldURLCount: {strconv.Itoa(math.MaxInt32)},
		"url0":                  {sampleURLMetadataWireForm(t, sampleURLMetadata(t, "url-a"))},
	}

	req, err := yacyproto.ParseTransferURLRequest(t.Context(), form)
	if err != nil {
		t.Fatalf("ParseTransferURLRequest: %v", err)
	}
	if len(req.URLs) != 1 {
		t.Fatalf("URLs = %d, want 1", len(req.URLs))
	}
}

func TestParseTransferURLRequestStopsAtRowsThatInflatePastTheLimit(t *testing.T) {
	t.Parallel()

	const messageMaxPlainBytes = 4 << 20
	row := strings.Replace(
		yacyURLMetadataRow,
		"descr=b|RXhhbXBsZQ",
		"descr="+gzipFramed(t, strings.Repeat("x", gzipBombPlainBytes)),
		1,
	)
	form := url.Values{yacyproto.FieldURLCount: {"8"}}
	for i := range 8 {
		form.Set("url"+strconv.Itoa(i), row)
	}

	req, err := yacyproto.ParseTransferURLRequest(t.Context(), form)
	if err != nil {
		t.Fatalf("ParseTransferURLRequest: %v", err)
	}
	inflatedBytes := 0
	for _, metadata := range req.URLs {
		inflatedBytes += len(metadata.Title)
	}
	if len(req.URLs) == 0 || inflatedBytes > messageMaxPlainBytes {
		t.Fatalf(
			"URLs = %d with %d title bytes, want at least 1 and at most %d bytes",
			len(req.URLs), inflatedBytes, messageMaxPlainBytes,
		)
	}
}

func TestParseTransferURLResponseRejectsUnknownResult(t *testing.T) {
	t.Parallel()

	fields := yacyproto.ResponseFields{yacyproto.FieldResult: "later"}
	if _, err := yacyproto.ParseTransferURLResponse(fields); err == nil {
		t.Fatal("expected error for unknown transferURL result")
	}
}
