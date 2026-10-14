package yacyproto_test

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func TestMessageRequestRoundTrip(t *testing.T) {
	t.Parallel()

	req := yacyproto.MessageRequest{
		NetworkName: yacyproto.DefaultNetwork,
		YouAre:      sampleHash(t, "alpha"),
		Iam:         sampleHash(t, "beta"),
		Key:         "salt",
		MyTime:      time.Date(2026, time.October, 14, 12, 0, 0, 0, time.UTC),
		Process:     yacyproto.ProcessPost,
		MySeed:      yacymodel.Some(sampleSeed(t, "beta", "peer-b")),
		Subject:     "your peer",
		Body:        "sends me spam",
	}

	got, err := yacyproto.ParseMessageRequest(t.Context(), req.Form())
	if err != nil {
		t.Fatalf("ParseMessageRequest: %v", err)
	}

	if !reflect.DeepEqual(got, req) {
		t.Fatalf("round-trip mismatch:\n got %#v\nwant %#v", got, req)
	}
}

func TestParseMessageRequestKeepsTextAsSent(t *testing.T) {
	t.Parallel()

	form := url.Values{
		yacyproto.FieldProcess: {string(yacyproto.ProcessPost)},
		yacyproto.FieldSubject: {"hi"},
		yacyproto.FieldMessage: {"b|not base64"},
	}

	got, err := yacyproto.ParseMessageRequest(t.Context(), form)
	if err != nil {
		t.Fatalf("ParseMessageRequest: %v", err)
	}

	if got.Subject != "hi" || got.Body != "b|not base64" {
		t.Fatalf("subject, body = %q, %q; want them as sent", got.Subject, got.Body)
	}
}

func TestParseMessageRequestAsksPermissionWithoutProcess(t *testing.T) {
	t.Parallel()

	got, err := yacyproto.ParseMessageRequest(t.Context(), url.Values{})
	if err != nil {
		t.Fatalf("ParseMessageRequest: %v", err)
	}

	if got.Process != yacyproto.ProcessPermission {
		t.Fatalf("process = %q, want %q", got.Process, yacyproto.ProcessPermission)
	}
}

func TestParseMessageRequestRejectsUnknownProcess(t *testing.T) {
	t.Parallel()

	form := url.Values{yacyproto.FieldProcess: {"delete"}}
	if _, err := yacyproto.ParseMessageRequest(t.Context(), form); err == nil {
		t.Fatal("expected error for unknown process")
	}
}

func TestParseMessageRequestRejectsMalformedSeed(t *testing.T) {
	t.Parallel()

	form := url.Values{yacyproto.FieldMySeed: {"b|%%%"}}
	if _, err := yacyproto.ParseMessageRequest(t.Context(), form); err == nil {
		t.Fatal("expected error for malformed seed")
	}
}

func TestMessageResponseRoundTrip(t *testing.T) {
	t.Parallel()

	resp := yacyproto.MessageResponse{
		ResponseHeader: yacyproto.ResponseHeader{Version: "1.0", Uptime: 7},
		Response:       yacyproto.MessageResponseWelcome,
		MessageSize:    10240,
	}

	fields := resp.Encode()
	yacyproto.InjectResponseHeader(fields, resp.Version, resp.Uptime)
	got, err := yacyproto.ParseMessageResponse(fields)
	if err != nil {
		t.Fatalf("ParseMessageResponse: %v", err)
	}

	if !reflect.DeepEqual(got, resp) {
		t.Fatalf("round-trip mismatch:\n got %#v\nwant %#v", got, resp)
	}
}

func TestParseMessageResponseRejectsBadMessageSize(t *testing.T) {
	t.Parallel()

	fields := yacyproto.ResponseFields{yacyproto.FieldMessageSize: "large"}
	if _, err := yacyproto.ParseMessageResponse(fields); err == nil {
		t.Fatal("expected error for malformed message size")
	}
}
