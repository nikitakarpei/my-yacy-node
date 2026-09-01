package urlmeta_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type stubRuntimeStatus struct{}

func (stubRuntimeStatus) Version(context.Context) string { return "1.0" }

func (stubRuntimeStatus) Uptime(context.Context) int { return 0 }

func muxWith(t *testing.T, receiver urlmeta.URLReceiver) *http.ServeMux {
	t.Helper()

	mux := http.NewServeMux()
	router := httpguard.NewWireRouter(mux, httpguard.WireGate{
		Guard: httpguard.NewRequestGuard(
			httpguard.DefaultMaxBodyBytes,
			httpguard.DefaultRequestTimeout,
		),
		Respond: httpguard.NewWireResponder(stubRuntimeStatus{}),
		Address: httpguard.NewClientAddressResolver(nil),
	})
	urlmeta.MountTransferURL(router, localIdentity(), receiver)

	return mux
}

func transferURL(
	t *testing.T,
	mux *http.ServeMux,
	req yacyproto.TransferURLRequest,
) yacyproto.TransferURLResponse {
	t.Helper()

	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		yacyproto.PathTransferURL,
		strings.NewReader(req.Form().Encode()),
	)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	mux.ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %q", rec.Code, rec.Body.String())
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	resp, err := yacyproto.ParseTransferURLResponse(yacyproto.ParseMessage(string(body)))
	if err != nil {
		t.Fatalf("ParseTransferURLResponse: %v", err)
	}

	return resp
}

func TestTransferURLStoresAndAnswers(t *testing.T) {
	v, module := openModule(t, 0)
	mux := muxWith(t, module.Receiver)

	resp := transferURL(t, mux, yacyproto.TransferURLRequest{
		NetworkName: "freeworld",
		YouAre:      localIdentity().Hash,
		URLCount:    1,
		URLs:        []yacymodel.URLMetadata{urlMetadata("a")},
	})

	if resp.Result != yacyproto.TransferURLResult(yacyproto.ResultOK) {
		t.Fatalf("Result = %q, want ok", resp.Result)
	}

	if count := storedURLCount(t, v, module.Directory); count != 1 {
		t.Fatalf("Count = %d, want the transferred url stored", count)
	}
}

func TestTransferURLRejectsWrongNetwork(t *testing.T) {
	_, module := openModule(t, 0)
	mux := muxWith(t, module.Receiver)

	resp := transferURL(t, mux, yacyproto.TransferURLRequest{
		NetworkName: "othernetwork",
		YouAre:      localIdentity().Hash,
		URLCount:    1,
		URLs:        []yacymodel.URLMetadata{urlMetadata("a")},
	})

	if resp.Result != yacyproto.TransferURLResult(yacyproto.ResultWrongTarget) {
		t.Fatalf("Result = %q, want wrong target", resp.Result)
	}
}
