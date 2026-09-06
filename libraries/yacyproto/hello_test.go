package yacyproto_test

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func TestHelloRequestRoundTrip(t *testing.T) {
	t.Parallel()

	req := yacyproto.HelloRequest{
		NetworkName: yacyproto.DefaultNetwork,
		Key:         "salt",
		Seed:        sampleSeed(t, "alpha", "peer-a"),
		Count:       50,
		Iam:         sampleHash(t, "alpha"),
		MagicMD5:    yacyproto.MagicMD5("k", "iam", "ess"),
		MyTime:      time.Date(2026, time.June, 17, 12, 0, 0, 0, time.UTC),
	}

	got, err := yacyproto.ParseHelloRequest(t.Context(), req.Form())
	if err != nil {
		t.Fatalf("ParseHelloRequest: %v", err)
	}

	if !reflect.DeepEqual(got, req) {
		t.Fatalf("round-trip mismatch:\n got %#v\nwant %#v", got, req)
	}
}

func TestHelloResponseRoundTrip(t *testing.T) {
	t.Parallel()

	resp := yacyproto.HelloResponse{
		ResponseHeader: yacyproto.ResponseHeader{Version: "1.0", Uptime: 42},
		YourIP:         "203.0.113.7",
		YourType:       yacymodel.Some(yacymodel.PeerSenior),
		MyTime:         time.Date(2026, time.June, 17, 12, 0, 1, 0, time.UTC),
		Message:        "ok",
		Seeds: []yacymodel.Seed{
			sampleSeed(t, "alpha", "peer-self"),
			sampleSeed(t, "beta", "peer-b"),
		},
	}

	msg := resp.Encode()
	yacyproto.InjectResponseHeader(msg, resp.Version, resp.Uptime)
	got, err := yacyproto.ParseHelloResponse(t.Context(), msg)
	if err != nil {
		t.Fatalf("ParseHelloResponse: %v", err)
	}

	if !reflect.DeepEqual(got, resp) {
		t.Fatalf("round-trip mismatch:\n got %#v\nwant %#v", got, resp)
	}
}

func TestParseHelloRequestRejectsBadMyTime(t *testing.T) {
	t.Parallel()

	form := url.Values{yacyproto.FieldMyTime: {"yesterday"}}
	if _, err := yacyproto.ParseHelloRequest(t.Context(), form); err == nil {
		t.Fatal("expected error for malformed mytime")
	}
}

func TestParseHelloRequestRejectsBadIam(t *testing.T) {
	t.Parallel()

	form := url.Values{yacyproto.FieldIam: {"short"}}
	if _, err := yacyproto.ParseHelloRequest(t.Context(), form); err == nil {
		t.Fatal("expected error for malformed iam hash")
	}
}

func TestParseHelloResponseRejectsBadPeerType(t *testing.T) {
	t.Parallel()

	msg := yacyproto.Message{yacyproto.FieldYourType: "overlord"}
	if _, err := yacyproto.ParseHelloResponse(t.Context(), msg); err == nil {
		t.Fatal("expected error for unknown peer type")
	}
}

func TestParseHelloResponseDropsMalformedKnownSeed(t *testing.T) {
	t.Parallel()

	own := sampleSeed(t, "alpha", "peer-self")
	known := sampleSeed(t, "gamma", "peer-c")
	msg := yacyproto.Message{
		"seed0": seedWireForm(own),
		"seed1": "not-a-valid-seed",
		"seed2": seedWireForm(known),
	}

	got, err := yacyproto.ParseHelloResponse(t.Context(), msg)
	if err != nil {
		t.Fatalf("ParseHelloResponse: %v", err)
	}

	want := []yacymodel.Seed{own, known}
	if !reflect.DeepEqual(got.Seeds, want) {
		t.Fatalf("Seeds:\n got %#v\nwant %#v", got.Seeds, want)
	}
}

func TestParseHelloResponseRejectsMalformedOwnSeed(t *testing.T) {
	t.Parallel()

	msg := yacyproto.Message{"seed0": "not-a-valid-seed"}
	if _, err := yacyproto.ParseHelloResponse(t.Context(), msg); err == nil {
		t.Fatal("expected error for malformed own seed")
	}
}
