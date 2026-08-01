package rwidistribution

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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

func courierSeed(t testing.TB, server *httptest.Server) yacymodel.Seed {
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

	return yacymodel.Seed{
		Hash:           courierHash("peer"),
		PrimaryAddress: yacymodel.Some(parsedHost),
		Port:           yacymodel.Some(port),
	}
}

func courierEndpoint(t testing.TB, server *httptest.Server) string {
	t.Helper()

	endpoint, ok := courierSeed(t, server).NetworkAddress()
	if !ok {
		t.Fatalf("courier seed has no network address")
	}

	return endpoint
}

func openCourierHarness(t *testing.T, server *httptest.Server) httpPostingCourier {
	t.Helper()

	return httpPostingCourier{
		exchange:    peerMessageExchange{client: server.Client()},
		networkName: "freeworld",
		self:        courierHash("self"),
	}
}

func rwiResponder(resp yacyproto.TransferRWIResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(resp.Encode().Encode()))
	}))
}

func singlePostingOffer(peer yacymodel.Seed) postingOffer {
	return postingOffer{
		Peer:     peer,
		Postings: []yacymodel.RWIPosting{fakePosting(yacymodel.WordHash("w1"), urlHash("u1"))},
	}
}

func TestOfferReportsAcceptedOnOK(t *testing.T) {
	server := rwiResponder(yacyproto.TransferRWIResponse{Result: yacyproto.ResultOK})
	defer server.Close()

	courier := openCourierHarness(t, server)

	receipt := courier.Offer(
		context.Background(),
		courierEndpoint(t, server),
		singlePostingOffer(courierSeed(t, server)),
	)
	if receipt.Outcome != postingOfferAccepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingOfferAccepted)
	}
}

func TestOfferReturnsUnknownURLsWithoutActingOnThem(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	unknown := fakePosting(word, url).URLHash
	server := rwiResponder(yacyproto.TransferRWIResponse{
		Result:     yacyproto.ResultOK,
		UnknownURL: []yacymodel.URLHash{unknown},
	})
	defer server.Close()

	courier := openCourierHarness(t, server)

	receipt := courier.Offer(
		context.Background(),
		courierEndpoint(t, server),
		singlePostingOffer(courierSeed(t, server)),
	)
	if receipt.Outcome != postingOfferAccepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingOfferAccepted)
	}
	if len(receipt.URLsUnknownToPeer) != 1 || receipt.URLsUnknownToPeer[0] != unknown {
		t.Fatalf("URLsUnknownToPeer = %v, want [%v]", receipt.URLsUnknownToPeer, unknown)
	}
}

func TestOfferReportsDeferredWithPeerPause(t *testing.T) {
	server := rwiResponder(yacyproto.TransferRWIResponse{
		Result: yacyproto.ResultBusy,
		Pause:  30 * time.Second,
	})
	defer server.Close()

	courier := openCourierHarness(t, server)

	receipt := courier.Offer(
		context.Background(),
		courierEndpoint(t, server),
		singlePostingOffer(courierSeed(t, server)),
	)
	if receipt.Outcome != postingOfferDeferred {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingOfferDeferred)
	}
	if receipt.RetryAfter != 30*time.Second {
		t.Fatalf("RetryAfter = %v, want 30s", receipt.RetryAfter)
	}
}

func TestOfferReportsOverloadedPeer(t *testing.T) {
	server := rwiResponder(yacyproto.TransferRWIResponse{Result: yacyproto.ResultTooHighLoad})
	defer server.Close()

	courier := openCourierHarness(t, server)

	receipt := courier.Offer(
		context.Background(),
		courierEndpoint(t, server),
		singlePostingOffer(courierSeed(t, server)),
	)
	if receipt.Outcome != postingOfferOverloaded {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingOfferOverloaded)
	}
}

func TestOfferReportsRefusalOnUnexpectedResult(t *testing.T) {
	server := rwiResponder(yacyproto.TransferRWIResponse{Result: yacyproto.ResultMissingIndexes})
	defer server.Close()

	courier := openCourierHarness(t, server)

	receipt := courier.Offer(
		context.Background(),
		courierEndpoint(t, server),
		singlePostingOffer(courierSeed(t, server)),
	)
	if receipt.Outcome != postingOfferRefused {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingOfferRefused)
	}
}

func TestOfferReportsUnreachablePeerOnTransportFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	courier := openCourierHarness(t, server)

	receipt := courier.Offer(
		context.Background(),
		courierEndpoint(t, server),
		singlePostingOffer(courierSeed(t, server)),
	)
	if receipt.Outcome != postingOfferUnreachable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingOfferUnreachable)
	}
}
