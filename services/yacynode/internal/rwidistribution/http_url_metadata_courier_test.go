package rwidistribution

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func urlResponder(resp yacyproto.TransferURLResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(resp.Encode().Encode()))
	}))
}

func openURLMetadataCourierHarness(t *testing.T, server *httptest.Server) httpURLMetadataCourier {
	t.Helper()

	return httpURLMetadataCourier{
		exchange:    peerwire.NewMessageExchange(server.Client()),
		networkName: "freeworld",
		self:        courierHash("self"),
	}
}

func singleURLMetadata() []yacymodel.URLMetadata {
	return []yacymodel.URLMetadata{{Address: "http://example.com/u1"}}
}

func TestDeliverReportsAcceptedOnOK(t *testing.T) {
	server := urlResponder(yacyproto.TransferURLResponse{Result: yacyproto.ResultOK})
	defer server.Close()

	courier := openURLMetadataCourierHarness(t, server)

	receipt := courier.Deliver(
		context.Background(), courierEndpoint(t, server), courierHash("peer"), singleURLMetadata(),
	)
	if receipt.Outcome != urlMetadataAccepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlMetadataAccepted)
	}
}

func TestDeliverReportsAcceptedWithRejectedURLs(t *testing.T) {
	rejected := urlHash("u1")
	server := urlResponder(yacyproto.TransferURLResponse{
		Result:   yacyproto.ResultOK,
		ErrorURL: []yacymodel.URLHash{rejected},
	})
	defer server.Close()

	courier := openURLMetadataCourierHarness(t, server)

	receipt := courier.Deliver(
		context.Background(), courierEndpoint(t, server), courierHash("peer"), singleURLMetadata(),
	)
	if receipt.Outcome != urlMetadataAccepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlMetadataAccepted)
	}
	if len(receipt.URLsRejected) != 1 || receipt.URLsRejected[0] != rejected {
		t.Fatalf("URLsRejected = %v, want [%v]", receipt.URLsRejected, rejected)
	}
}

func TestDeliverReportsDeferredOnErrorNotGranted(t *testing.T) {
	server := urlResponder(yacyproto.TransferURLResponse{Result: yacyproto.ResultErrorNotGranted})
	defer server.Close()

	courier := openURLMetadataCourierHarness(t, server)

	receipt := courier.Deliver(
		context.Background(), courierEndpoint(t, server), courierHash("peer"), singleURLMetadata(),
	)
	if receipt.Outcome != urlMetadataDeferred {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlMetadataDeferred)
	}
}

func TestDeliverReportsRefusedOnWrongTarget(t *testing.T) {
	server := urlResponder(yacyproto.TransferURLResponse{Result: yacyproto.ResultWrongTarget})
	defer server.Close()

	courier := openURLMetadataCourierHarness(t, server)

	receipt := courier.Deliver(
		context.Background(), courierEndpoint(t, server), courierHash("peer"), singleURLMetadata(),
	)
	if receipt.Outcome != urlMetadataRefused {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlMetadataRefused)
	}
}

func TestDeliverReportsUnreachableOnTransportFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	courier := openURLMetadataCourierHarness(t, server)

	receipt := courier.Deliver(
		context.Background(), courierEndpoint(t, server), courierHash("peer"), singleURLMetadata(),
	)
	if receipt.Outcome != urlMetadataUnreachable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlMetadataUnreachable)
	}
}
