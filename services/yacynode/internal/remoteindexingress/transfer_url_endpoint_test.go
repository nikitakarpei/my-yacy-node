package remoteindexingress_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func urlMetadata(t *testing.T, seed string) yacymodel.URLMetadata {
	t.Helper()

	address := "http://example.com/" + seed
	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return yacymodel.URLMetadata{Hash: hash, Address: address}
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

	resp, err := yacyproto.ParseTransferURLResponse(yacyproto.ParseResponseFields(string(body)))
	if err != nil {
		t.Fatalf("ParseTransferURLResponse: %v", err)
	}

	return resp
}

func TestTransferURLHandsTheMetadataOverWithItsSenderAndAnswers(t *testing.T) {
	receiver := &recordingURLReceiver{}
	mux := mountedMux(localIdentity(), &recordingReceiver{}, receiver)

	resp := transferURL(t, mux, yacyproto.TransferURLRequest{
		NetworkName: "freeworld",
		YouAre:      localIdentity().Hash,
		URLCount:    1,
		URLs:        []yacymodel.URLMetadata{urlMetadata(t, "a")},
	})

	if resp.Result != yacyproto.TransferURLResult(yacyproto.ResultOK) {
		t.Fatalf("Result = %q, want ok", resp.Result)
	}
	if len(receiver.received) != 1 || receiver.senders[0] != rwiorigin.Of("192.0.2.1") {
		t.Fatalf("received = %v from %v, want the url from the request's network",
			receiver.received, receiver.senders)
	}
}

func TestTransferURLRejectsWrongNetwork(t *testing.T) {
	receiver := &recordingURLReceiver{}
	mux := mountedMux(localIdentity(), &recordingReceiver{}, receiver)

	resp := transferURL(t, mux, yacyproto.TransferURLRequest{
		NetworkName: "othernetwork",
		YouAre:      localIdentity().Hash,
		URLCount:    1,
		URLs:        []yacymodel.URLMetadata{urlMetadata(t, "a")},
	})

	if resp.Result != yacyproto.TransferURLResult(yacyproto.ResultWrongTarget) ||
		len(receiver.received) != 0 {
		t.Fatalf("Result = %q, received = %v, want wrong target and nothing handed over",
			resp.Result, receiver.received)
	}
}

func TestTransferURLRefusesWhenRemoteIndexIsNotAccepted(t *testing.T) {
	identity := localIdentity()
	identity.Capabilities.AcceptRemoteIndex = false
	receiver := &recordingURLReceiver{}
	mux := mountedMux(identity, &recordingReceiver{}, receiver)

	resp := transferURL(t, mux, yacyproto.TransferURLRequest{
		NetworkName: "freeworld",
		YouAre:      identity.Hash,
		URLCount:    1,
		URLs:        []yacymodel.URLMetadata{urlMetadata(t, "a")},
	})

	if resp.Result != yacyproto.ResultErrorNotGranted || len(receiver.received) != 0 {
		t.Fatalf("Result = %q, received = %v, want error not granted and nothing handed over",
			resp.Result, receiver.received)
	}
}
