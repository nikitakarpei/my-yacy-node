package httpguard_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type echoResponse struct {
	addr string
}

func (r echoResponse) Encode() yacyproto.Message {
	return yacyproto.Message{"yourip": r.addr}
}

type echoFeed struct {
	addr string
}

func (f echoFeed) Encode() []byte {
	return []byte("<rss>" + f.addr + "</rss>")
}

func TestServeFeedWritesTheFeedAsXML(t *testing.T) {
	handler := httpguard.ServeFeed(
		testGate(),
		yacyproto.URLMetadataEndpointMethods,
		func(ctx context.Context, _ url.Values) (echoFeed, error) {
			return echoFeed{addr: httpguard.RemoteAddr(ctx)}, nil
		},
		func(_ context.Context, feed echoFeed) (echoFeed, error) {
			return feed, nil
		},
	)

	rec := httptest.NewRecorder()
	req := postForm(yacyproto.PathURLMetadata)
	req.RemoteAddr = "203.0.113.9:5000"
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "<rss>203.0.113.9</rss>" {
		t.Fatalf("body = %q, want the feed with the resolved remote address", got)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/xml") {
		t.Fatalf("content type = %q, want xml", got)
	}
}

func testGate() httpguard.WireGate {
	return httpguard.WireGate{
		Guard:   testGuard(),
		Respond: httpguard.NewWireResponder(stubStatus{}),
		Address: httpguard.NewClientAddressResolver(nil),
	}
}

func postForm(target string) *http.Request {
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		target,
		strings.NewReader("a=b"),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return req
}

func TestServeMessageWritesResponseWithRemoteAddr(t *testing.T) {
	handler := httpguard.ServeMessage(
		testGate(),
		yacyproto.TransferURLEndpointMethods,
		func(ctx context.Context, _ url.Values) (echoResponse, error) {
			return echoResponse{addr: httpguard.RemoteAddr(ctx)}, nil
		},
		func(_ context.Context, resp echoResponse) (echoResponse, error) {
			return resp, nil
		},
	)

	rec := httptest.NewRecorder()
	req := postForm(yacyproto.PathTransferURL)
	req.RemoteAddr = "203.0.113.9:5000"
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "203.0.113.9") {
		t.Fatalf("body = %q, want resolved remote address", rec.Body.String())
	}
}

func TestServeMessageMapsParseErrorToBadRequest(t *testing.T) {
	handler := httpguard.ServeMessage(
		testGate(),
		yacyproto.TransferURLEndpointMethods,
		func(context.Context, url.Values) (echoResponse, error) {
			return echoResponse{}, errors.New("bad form")
		},
		func(_ context.Context, resp echoResponse) (echoResponse, error) {
			return resp, nil
		},
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, postForm(yacyproto.PathTransferURL))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestServeMessageMapsServeErrorToInternal(t *testing.T) {
	handler := httpguard.ServeMessage(
		testGate(),
		yacyproto.TransferURLEndpointMethods,
		func(context.Context, url.Values) (echoResponse, error) {
			return echoResponse{}, nil
		},
		func(context.Context, echoResponse) (echoResponse, error) {
			return echoResponse{}, errors.New("boom")
		},
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, postForm(yacyproto.PathTransferURL))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestServeMessageRejectsDisallowedMethod(t *testing.T) {
	handler := httpguard.ServeMessage(
		testGate(),
		yacyproto.TransferURLEndpointMethods,
		func(context.Context, url.Values) (echoResponse, error) {
			return echoResponse{}, nil
		},
		func(_ context.Context, resp echoResponse) (echoResponse, error) {
			return resp, nil
		},
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		yacyproto.PathTransferURL,
		nil,
	))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}
