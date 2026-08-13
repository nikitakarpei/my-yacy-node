package urlmetadatacourier

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const courierHashFiller = "AAAAAAAAAAAA"

func courierHash(base string) yacymodel.Hash {
	padded := base + courierHashFiller
	hash, err := yacymodel.ParseHash(padded[:yacymodel.HashLength])
	if err != nil {
		panic(err)
	}

	return hash
}

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func courierEndpoint(t testing.TB, server *httptest.Server) string {
	t.Helper()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	host, portRaw, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	parsedHost, err := yacymodel.ParseHost(host)
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}
	port, err := yacymodel.ParsePort(portRaw)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	endpoint, ok := yacymodel.Seed{
		Hash:           courierHash("peer"),
		PrimaryAddress: yacymodel.Some(parsedHost),
		Port:           yacymodel.Some(port),
	}.NetworkAddress()
	if !ok {
		t.Fatalf("courier seed has no network address")
	}

	return endpoint
}

func urlResponder(resp yacyproto.TransferURLResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(resp.Encode().Encode()))
	}))
}

func openURLMetadataCourierHarness(t *testing.T, server *httptest.Server) httpCourier {
	t.Helper()

	return httpCourier{
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
	if receipt.Outcome != Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, Accepted)
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
	if receipt.Outcome != Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, Accepted)
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
	if receipt.Outcome != Deferred {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, Deferred)
	}
}

func TestDeliverReportsRefusedOnWrongTarget(t *testing.T) {
	server := urlResponder(yacyproto.TransferURLResponse{Result: yacyproto.ResultWrongTarget})
	defer server.Close()

	courier := openURLMetadataCourierHarness(t, server)

	receipt := courier.Deliver(
		context.Background(), courierEndpoint(t, server), courierHash("peer"), singleURLMetadata(),
	)
	if receipt.Outcome != Refused {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, Refused)
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
	if receipt.Outcome != Unreachable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, Unreachable)
	}
}
