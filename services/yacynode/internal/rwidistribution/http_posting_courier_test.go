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

func openCourierHarness(t *testing.T, server *httptest.Server) httpPostingCourier {
	t.Helper()

	return httpPostingCourier{
		client:      server.Client(),
		networkName: "freeworld",
		self:        courierHash("self"),
		urls:        fakeURLDirectory{},
	}
}

type fakeURLDirectory struct {
	metadata map[yacymodel.URLHash]yacymodel.URLMetadata
}

func (f fakeURLDirectory) MetadataByHash(
	_ context.Context,
	hashes []yacymodel.URLHash,
) ([]yacymodel.URLMetadata, error) {
	found := make([]yacymodel.URLMetadata, 0, len(hashes))
	for _, hash := range hashes {
		if entry, ok := f.metadata[hash]; ok {
			found = append(found, entry)
		}
	}

	return found, nil
}

func (fakeURLDirectory) MissingURLs(
	context.Context,
	[]yacymodel.URLHash,
) ([]yacymodel.URLHash, error) {
	return nil, nil
}

func (fakeURLDirectory) Count(context.Context) (int, error) { return 0, nil }

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

	receipt := courier.Offer(context.Background(), singlePostingOffer(courierSeed(t, server)))
	if receipt.Outcome != postingOfferAccepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingOfferAccepted)
	}
}

func TestOfferDeliversMetadataForUnknownURLs(t *testing.T) {
	var gotPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		if r.URL.Path == yacyproto.PathTransferRWI {
			word, url := yacymodel.WordHash("w1"), urlHash("u1")
			resp := yacyproto.TransferRWIResponse{
				Result:     yacyproto.ResultOK,
				UnknownURL: []yacymodel.URLHash{fakePosting(word, url).URLHash},
			}
			_, _ = w.Write([]byte(resp.Encode().Encode()))

			return
		}
		resp := yacyproto.TransferURLResponse{Result: yacyproto.ResultOK}
		_, _ = w.Write([]byte(resp.Encode().Encode()))
	}))
	defer server.Close()

	courier := openCourierHarness(t, server)
	posting := fakePosting(yacymodel.WordHash("w1"), urlHash("u1"))
	courier.urls = fakeURLDirectory{
		metadata: map[yacymodel.URLHash]yacymodel.URLMetadata{
			posting.URLHash: {Address: "http://example.com/u1"},
		},
	}

	receipt := courier.Offer(context.Background(), singlePostingOffer(courierSeed(t, server)))
	if receipt.Outcome != postingOfferAccepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingOfferAccepted)
	}
	if len(gotPaths) != 2 || gotPaths[0] != yacyproto.PathTransferRWI ||
		gotPaths[1] != yacyproto.PathTransferURL {
		t.Fatalf("paths = %v, want [transferRWI transferURL]", gotPaths)
	}
}

func TestOfferSkipsTransferURLWhenMetadataMissing(t *testing.T) {
	var gotPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		word, url := yacymodel.WordHash("w1"), urlHash("u1")
		resp := yacyproto.TransferRWIResponse{
			Result:     yacyproto.ResultOK,
			UnknownURL: []yacymodel.URLHash{fakePosting(word, url).URLHash},
		}
		_, _ = w.Write([]byte(resp.Encode().Encode()))
	}))
	defer server.Close()

	courier := openCourierHarness(t, server)

	receipt := courier.Offer(context.Background(), singlePostingOffer(courierSeed(t, server)))
	if receipt.Outcome != postingOfferAccepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingOfferAccepted)
	}
	if len(gotPaths) != 1 || gotPaths[0] != yacyproto.PathTransferRWI {
		t.Fatalf("paths = %v, want only [transferRWI] when no metadata is found", gotPaths)
	}
}

func TestOfferReportsDeferredWithPeerPause(t *testing.T) {
	server := rwiResponder(yacyproto.TransferRWIResponse{
		Result: yacyproto.ResultBusy,
		Pause:  30 * time.Second,
	})
	defer server.Close()

	courier := openCourierHarness(t, server)

	receipt := courier.Offer(context.Background(), singlePostingOffer(courierSeed(t, server)))
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

	receipt := courier.Offer(context.Background(), singlePostingOffer(courierSeed(t, server)))
	if receipt.Outcome != postingOfferOverloaded {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingOfferOverloaded)
	}
}

func TestOfferReportsRefusalOnUnexpectedResult(t *testing.T) {
	server := rwiResponder(yacyproto.TransferRWIResponse{Result: yacyproto.ResultMissingIndexes})
	defer server.Close()

	courier := openCourierHarness(t, server)

	receipt := courier.Offer(context.Background(), singlePostingOffer(courierSeed(t, server)))
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

	receipt := courier.Offer(context.Background(), singlePostingOffer(courierSeed(t, server)))
	if receipt.Outcome != postingOfferUnreachable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingOfferUnreachable)
	}
}

func TestOfferReportsUnaddressablePeer(t *testing.T) {
	server := httptest.NewServer(nil)
	defer server.Close()

	courier := openCourierHarness(t, server)
	offer := singlePostingOffer(yacymodel.Seed{Hash: courierHash("peer")})

	receipt := courier.Offer(context.Background(), offer)
	if receipt.Outcome != postingOfferUnaddressable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingOfferUnaddressable)
	}
}
