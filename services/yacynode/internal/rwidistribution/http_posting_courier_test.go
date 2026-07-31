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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
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

type recordingRoster struct {
	fakeRoster
	unreachable []yacymodel.Hash
}

func (r *recordingRoster) ConfirmUnreachable(_ context.Context, peer yacymodel.Hash) {
	r.unreachable = append(r.unreachable, peer)
}

func openCourierHarness(
	t *testing.T,
	server *httptest.Server,
) (*replicaLedger, *recordingRoster, *fakePostingOfferCycleObserver, httpPostingCourier) {
	t.Helper()

	v, err := memvault.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	ledger, err := openReplicaLedger(v)
	if err != nil {
		t.Fatalf("openReplicaLedger: %v", err)
	}

	roster := &recordingRoster{}
	observer := newFakePostingOfferCycleObserver()
	courier := httpPostingCourier{
		client:      server.Client(),
		networkName: "freeworld",
		self:        courierHash("self"),
		roster:      roster,
		ledger:      ledger,
		urls:        fakeURLDirectory{},
		observer:    observer,
	}

	return ledger, roster, observer, courier
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

func TestOfferRecordsReplicaOnOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := yacyproto.TransferRWIResponse{Result: yacyproto.ResultOK}
		_, _ = w.Write([]byte(resp.Encode().Encode()))
	}))
	defer server.Close()

	ledger, roster, observer, courier := openCourierHarness(t, server)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := courierSeed(t, server)
	offer := postingOffer{
		Peer:     peer,
		Postings: []yacymodel.RWIPosting{fakePosting(word, url)},
	}

	outcome := courier.Offer(context.Background(), offer)
	if !outcome.Accepted {
		t.Fatalf("outcome = %+v, want Accepted", outcome)
	}
	if len(roster.unreachable) != 0 {
		t.Fatalf("unreachable = %v, want none", roster.unreachable)
	}
	if observer.postingsOffered[string(yacyproto.ResultOK)] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for result %q",
			observer.postingsOffered,
			yacyproto.ResultOK,
		)
	}

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 1 || replicas[0] != peer.Hash {
		t.Fatalf("replicas = %v, want [%v]", replicas, peer.Hash)
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

	_, _, _, courier := openCourierHarness(t, server)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	posting := fakePosting(word, url)
	courier.urls = fakeURLDirectory{
		metadata: map[yacymodel.URLHash]yacymodel.URLMetadata{
			posting.URLHash: {Address: "http://example.com/u1"},
		},
	}
	peer := courierSeed(t, server)
	offer := postingOffer{
		Peer:     peer,
		Postings: []yacymodel.RWIPosting{posting},
	}

	if outcome := courier.Offer(context.Background(), offer); !outcome.Accepted {
		t.Fatalf("outcome = %+v, want Accepted", outcome)
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

	_, _, _, courier := openCourierHarness(t, server)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := courierSeed(t, server)
	offer := postingOffer{
		Peer:     peer,
		Postings: []yacymodel.RWIPosting{fakePosting(word, url)},
	}

	if outcome := courier.Offer(context.Background(), offer); !outcome.Accepted {
		t.Fatalf("outcome = %+v, want Accepted", outcome)
	}
	if len(gotPaths) != 1 || gotPaths[0] != yacyproto.PathTransferRWI {
		t.Fatalf("paths = %v, want only [transferRWI] when no metadata is found", gotPaths)
	}
}

func TestOfferHonoursBusyPause(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := yacyproto.TransferRWIResponse{Result: yacyproto.ResultBusy, Pause: 30 * time.Second}
		_, _ = w.Write([]byte(resp.Encode().Encode()))
	}))
	defer server.Close()

	ledger, roster, observer, courier := openCourierHarness(t, server)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := courierSeed(t, server)
	offer := postingOffer{
		Peer:     peer,
		Postings: []yacymodel.RWIPosting{fakePosting(word, url)},
	}

	outcome := courier.Offer(context.Background(), offer)
	if outcome.Accepted {
		t.Fatal("outcome should not be accepted when busy")
	}
	if outcome.RetryAfter.Seconds() != 30 {
		t.Fatalf("RetryAfter = %v, want 30s", outcome.RetryAfter)
	}
	if len(roster.unreachable) != 0 {
		t.Fatalf("unreachable = %v, want none when busy", roster.unreachable)
	}
	if observer.postingsOffered[string(yacyproto.ResultBusy)] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for result %q",
			observer.postingsOffered,
			yacyproto.ResultBusy,
		)
	}

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none when busy", replicas)
	}
}

func TestOfferKeepsLoadedPeerReachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := yacyproto.TransferRWIResponse{Result: yacyproto.ResultTooHighLoad}
		_, _ = w.Write([]byte(resp.Encode().Encode()))
	}))
	defer server.Close()

	_, roster, observer, courier := openCourierHarness(t, server)
	peer := courierSeed(t, server)
	offer := postingOffer{
		Peer: peer,
		Postings: []yacymodel.RWIPosting{
			fakePosting(yacymodel.WordHash("w1"), urlHash("u1")),
		},
	}

	outcome := courier.Offer(context.Background(), offer)
	if outcome.Accepted {
		t.Fatal("outcome should not be accepted")
	}
	if len(roster.unreachable) != 0 {
		t.Fatalf("unreachable = %v, want none when the peer is loaded", roster.unreachable)
	}
	if observer.postingsOffered[string(yacyproto.ResultTooHighLoad)] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for result %q",
			observer.postingsOffered, yacyproto.ResultTooHighLoad,
		)
	}
}

func TestOfferMarksPeerUnreachableOnUnexpectedResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := yacyproto.TransferRWIResponse{Result: yacyproto.ResultMissingIndexes}
		_, _ = w.Write([]byte(resp.Encode().Encode()))
	}))
	defer server.Close()

	_, roster, observer, courier := openCourierHarness(t, server)
	peer := courierSeed(t, server)
	offer := postingOffer{
		Peer: peer,
		Postings: []yacymodel.RWIPosting{
			fakePosting(yacymodel.WordHash("w1"), urlHash("u1")),
		},
	}

	outcome := courier.Offer(context.Background(), offer)
	if outcome.Accepted {
		t.Fatal("outcome should not be accepted")
	}
	if len(roster.unreachable) != 1 || roster.unreachable[0] != peer.Hash {
		t.Fatalf("unreachable = %v, want [%v]", roster.unreachable, peer.Hash)
	}
	if observer.postingsOffered[string(yacyproto.ResultMissingIndexes)] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for result %q",
			observer.postingsOffered, yacyproto.ResultMissingIndexes,
		)
	}
}

func TestOfferMarksPeerUnreachableOnTransportFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, roster, observer, courier := openCourierHarness(t, server)
	peer := courierSeed(t, server)
	offer := postingOffer{
		Peer: peer,
		Postings: []yacymodel.RWIPosting{
			fakePosting(yacymodel.WordHash("w1"), urlHash("u1")),
		},
	}

	outcome := courier.Offer(context.Background(), offer)
	if outcome.Accepted {
		t.Fatal("outcome should not be accepted")
	}
	if len(roster.unreachable) != 1 || roster.unreachable[0] != peer.Hash {
		t.Fatalf("unreachable = %v, want [%v]", roster.unreachable, peer.Hash)
	}
	if observer.postingsOffered[resultError] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for result %q",
			observer.postingsOffered,
			resultError,
		)
	}
}

func TestOfferMarksPeerUnreachableWithoutNetworkAddress(t *testing.T) {
	_, roster, observer, courier := openCourierHarness(t, httptest.NewServer(nil))
	peer := yacymodel.Seed{Hash: courierHash("peer")}
	offer := postingOffer{
		Peer: peer,
		Postings: []yacymodel.RWIPosting{
			fakePosting(yacymodel.WordHash("w1"), urlHash("u1")),
		},
	}

	outcome := courier.Offer(context.Background(), offer)
	if outcome.Accepted {
		t.Fatal("outcome should not be accepted")
	}
	if len(roster.unreachable) != 1 || roster.unreachable[0] != peer.Hash {
		t.Fatalf("unreachable = %v, want [%v]", roster.unreachable, peer.Hash)
	}
	if observer.postingsOffered[resultUnreachable] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for result %q",
			observer.postingsOffered, resultUnreachable,
		)
	}
}
