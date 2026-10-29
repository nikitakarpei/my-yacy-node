package peeradmission_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peeradmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type helloRuntimeStatus struct{}

func (helloRuntimeStatus) Version(context.Context) string { return "1.0" }

func (helloRuntimeStatus) Uptime(context.Context) int { return 0 }

type stubReachability struct {
	guard             sync.Mutex
	seeds             []yacymodel.Seed
	refreshed         []yacymodel.Seed
	confirmedBaseURLs []peerwire.BaseURL
	answeredBaseURLs  map[yacymodel.Hash]peerwire.BaseURL
}

func (s *stubReachability) MostRecentlyReachablePeers(
	_ context.Context,
	limit int,
) []yacymodel.Seed {
	if limit <= 0 {
		return nil
	}
	if limit < len(s.seeds) {
		return s.seeds[:limit]
	}

	return s.seeds
}

func (s *stubReachability) ConfirmSecondhandSeed(
	_ context.Context,
	seed yacymodel.Seed,
	answeredBaseURL peerwire.BaseURL,
) {
	s.guard.Lock()
	defer s.guard.Unlock()

	s.refreshed = append(s.refreshed, seed)
	s.confirmedBaseURLs = append(s.confirmedBaseURLs, answeredBaseURL)
}

func (s *stubReachability) RecordOf(
	_ context.Context,
	peer yacymodel.Hash,
) (peerroster.PeerRecord, bool) {
	baseURL, found := s.answeredBaseURLs[peer]
	if !found {
		return peerroster.PeerRecord{}, false
	}

	return peerroster.PeerRecord{AnsweredBaseURL: yacymodel.Some(baseURL)}, true
}

var replyTime = time.Date(2026, time.June, 17, 12, 0, 0, 0, time.UTC)

type stoppedClock struct{}

func (stoppedClock) Now() time.Time { return replyTime }

func (stoppedClock) After(time.Duration, func()) (stop func()) { return func() {} }

func muxWithHello(
	t *testing.T,
	reachability *stubReachability,
	client *http.Client,
) *http.ServeMux {
	t.Helper()

	return muxWithHelloAt(t, reachability, client, stoppedClock{}, nil)
}

func muxWithHelloAt(
	t *testing.T,
	reachability *stubReachability,
	client *http.Client,
	clock peeradmission.Clock,
	peerTransports peerwire.PeerTransports,
) *http.ServeMux {
	mux := http.NewServeMux()
	router := httpguard.NewWireRouter(mux, httpguard.WireGate{
		Guard: httpguard.NewRequestGuard(
			httpguard.DefaultMaxBodyBytes,
			httpguard.DefaultRequestTimeout,
		),
		Respond: httpguard.NewWireResponder(helloRuntimeStatus{}),
		Address: httpguard.NewClientAddressResolver(nil),
	})

	peeradmission.MountHello(
		router,
		localPeer(),
		selfStatus(t),
		clock,
		reachability,
		peerwire.NewWalker(client, peerTransports, stoppedClock{}),
	)

	return mux
}

func helloRequest(network string, caller yacymodel.Seed, count int) yacyproto.HelloRequest {
	return yacyproto.HelloRequest{
		NetworkName: network,
		Seed:        caller,
		Iam:         caller.Hash,
		Count:       count,
	}
}

func serveHello(
	t *testing.T,
	mux *http.ServeMux,
	req yacyproto.HelloRequest,
) yacyproto.HelloResponse {
	t.Helper()

	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		yacyproto.PathHello,
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

	resp, err := yacyproto.ParseHelloResponse(
		context.Background(),
		yacyproto.ParseResponseFields(body),
	)
	if err != nil {
		t.Fatalf("ParseHelloResponse: %v", err)
	}

	return resp
}

func backPingServer(reachable bool) *httptest.Server {
	return httptest.NewServer(backPingHandler(reachable))
}

func backPingHandler(reachable bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !reachable {
			http.Error(w, "boom", http.StatusInternalServerError)

			return
		}
		resp := yacyproto.QueryResponse{Response: 3}
		_, _ = io.WriteString(w, resp.Encode().Encode())
	})
}

func TestHelloClassifiesReachableAddressedCallerAsSenior(t *testing.T) {
	srv := backPingServer(true)
	defer srv.Close()

	reachability := &stubReachability{
		seeds: []yacymodel.Seed{peerSeed(t, "trusted", "203.0.113.1", 8090)},
	}
	mux := muxWithHello(t, reachability, srv.Client())

	caller := reachableCallerSeed(t, srv.URL)
	resp := serveHello(t, mux, helloRequest("freeworld", caller, 20))

	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerSenior {
		t.Fatalf("YourType = %q, want senior", yourType)
	}
	if got := len(resp.Seeds); got != 2 {
		t.Fatalf("Seeds = %d, want 2 (self + trusted)", got)
	}
	if resp.Seeds[0].Hash != hashFor("self") {
		t.Fatalf("first seed = %q, want self", resp.Seeds[0].Hash)
	}
	if !reflect.DeepEqual(reachability.refreshed, []yacymodel.Seed{caller}) {
		t.Fatalf("refreshed = %v, want senior caller refreshed", reachability.refreshed)
	}
}

func TestHelloBackPingNamesThisNodeAsTheAsker(t *testing.T) {
	var askers []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		askers = append(askers, r.PostFormValue(yacyproto.FieldIam))
		resp := yacyproto.QueryResponse{Response: 3}
		_, _ = io.WriteString(w, resp.Encode().Encode())
	}))
	defer srv.Close()
	mux := muxWithHello(t, &stubReachability{}, srv.Client())

	serveHello(t, mux, helloRequest("freeworld", reachableCallerSeed(t, srv.URL), 0))

	if want := []string{localPeer().Hash.String()}; !slices.Equal(askers, want) {
		t.Fatalf("back-ping askers = %v, want %v", askers, want)
	}
}

type recordingPeerTransports struct {
	transport  http.RoundTripper
	identities []yacymodel.PeerIdentity
}

func (r *recordingPeerTransports) TransportFor(identity yacymodel.PeerIdentity) http.RoundTripper {
	r.identities = append(r.identities, identity)

	return r.transport
}

func TestHelloBackPingsASignedCallerOverHTTPSForItsIdentityAtItsSecurePort(t *testing.T) {
	srv := httptest.NewTLSServer(backPingHandler(true))
	defer srv.Close()
	peerTransports := &recordingPeerTransports{transport: srv.Client().Transport}
	reachability := &stubReachability{}
	mux := muxWithHelloAt(t, reachability, http.DefaultClient, stoppedClock{}, peerTransports)
	caller := signedCallerSeedWithSecurePort(t, srv.URL)

	resp := serveHello(t, mux, helloRequest("freeworld", caller, 0))

	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerSenior {
		t.Fatalf("YourType = %q, want senior", yourType)
	}
	if len(reachability.confirmedBaseURLs) != 1 ||
		reachability.confirmedBaseURLs[0].Scheme != peerwire.HTTPS {
		t.Errorf(
			"confirmed base URLs = %v, want one https base URL",
			reachability.confirmedBaseURLs,
		)
	}
	want := []yacymodel.PeerIdentity{peerkeytest.Key(0).Identity()}
	if !slices.Equal(peerTransports.identities, want) {
		t.Errorf(
			"peer transports asked for %v, want the caller's identity",
			peerTransports.identities,
		)
	}
}

func TestHelloClassifiesUnreachableCallerAsJunior(t *testing.T) {
	srv := backPingServer(false)
	defer srv.Close()

	reachability := &stubReachability{}
	mux := muxWithHello(t, reachability, srv.Client())

	resp := serveHello(
		t,
		mux,
		helloRequest("freeworld", reachableCallerSeed(t, srv.URL), 0),
	)
	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerJunior {
		t.Fatalf("YourType = %q, want junior", yourType)
	}
	if len(reachability.refreshed) != 0 {
		t.Fatalf("refreshed = %v, want no refresh for junior caller", reachability.refreshed)
	}
}

func TestHelloClassifiesAddresslessCallerAsJunior(t *testing.T) {
	reachability := &stubReachability{}
	mux := muxWithHello(t, reachability, http.DefaultClient)

	resp := serveHello(
		t,
		mux,
		helloRequest("freeworld", addresslessCallerSeed(t), 0),
	)
	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerJunior {
		t.Fatalf("YourType = %q, want junior for addressless caller", yourType)
	}
	if len(reachability.refreshed) != 0 {
		t.Fatalf("refreshed = %v, want no refresh for addressless caller", reachability.refreshed)
	}
}

func TestHelloOnForeignNetworkAnswersVirginWithoutSeeds(t *testing.T) {
	reachability := &stubReachability{
		seeds: []yacymodel.Seed{peerSeed(t, "trusted", "203.0.113.1", 8090)},
	}
	mux := muxWithHello(t, reachability, http.DefaultClient)

	resp := serveHello(
		t,
		mux,
		helloRequest("otherworld", unreachableCallerSeed(t), 20),
	)
	if got := len(resp.Seeds); got != 0 {
		t.Fatalf("Seeds = %d, want none for foreign network", got)
	}
	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerVirgin {
		t.Fatalf("YourType = %q, want virgin for foreign network", yourType)
	}
	if resp.Message != "not in my network" {
		t.Fatalf("Message = %q, want %q", resp.Message, "not in my network")
	}
	if !resp.MyTime.Equal(replyTime) {
		t.Fatalf("MyTime = %v, want %v", resp.MyTime, replyTime)
	}
}

func TestHelloOnOwnNetworkAnswersOwnTimeAndOkMessage(t *testing.T) {
	mux := muxWithHello(t, &stubReachability{}, http.DefaultClient)

	resp := serveHello(
		t,
		mux,
		helloRequest("freeworld", unreachableCallerSeed(t), 20),
	)
	if resp.Message != "ok" {
		t.Fatalf("Message = %q, want %q", resp.Message, "ok")
	}
	if !resp.MyTime.Equal(replyTime) {
		t.Fatalf("MyTime = %v, want %v", resp.MyTime, replyTime)
	}
}

func TestHelloClassifiesReachablePrincipalCallerAsPrincipal(t *testing.T) {
	srv := backPingServer(true)
	defer srv.Close()

	mux := muxWithHello(t, &stubReachability{}, srv.Client())

	caller := reachableCallerSeed(t, srv.URL)
	caller.PeerType = yacymodel.PeerPrincipal
	resp := serveHello(t, mux, helloRequest("freeworld", caller, 20))

	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerPrincipal {
		t.Fatalf("YourType = %q, want principal", yourType)
	}
}

func TestHelloLimitsKnownPeersToCount(t *testing.T) {
	reachability := &stubReachability{seeds: []yacymodel.Seed{
		peerSeed(t, "a", "203.0.113.1", 8090),
		peerSeed(t, "b", "203.0.113.2", 8090),
		peerSeed(t, "c", "203.0.113.3", 8090),
	}}
	mux := muxWithHello(t, reachability, http.DefaultClient)

	resp := serveHello(
		t,
		mux,
		helloRequest("freeworld", unreachableCallerSeed(t), 2),
	)

	if got := len(resp.Seeds); got != 3 {
		t.Fatalf("Seeds = %d, want 3 (self + two of three known)", got)
	}
	known := []yacymodel.Hash{hashFor("a"), hashFor("b"), hashFor("c")}
	for _, seed := range resp.Seeds[1:] {
		if !slices.Contains(known, seed.Hash) {
			t.Fatalf("known peer %q not from roster %v", seed.Hash, known)
		}
	}
}

func TestHelloCountZeroReturnsNoKnownPeers(t *testing.T) {
	reachability := &stubReachability{seeds: []yacymodel.Seed{
		peerSeed(t, "a", "203.0.113.1", 8090),
		peerSeed(t, "b", "203.0.113.2", 8090),
	}}
	mux := muxWithHello(t, reachability, http.DefaultClient)

	resp := serveHello(
		t,
		mux,
		helloRequest("freeworld", unreachableCallerSeed(t), 0),
	)
	if got := len(resp.Seeds); got != 1 {
		t.Fatalf("Seeds = %d, want 1 (self only)", got)
	}
}

func TestHelloCapsKnownPeersAtOneHundred(t *testing.T) {
	reachability := &stubReachability{}
	for index := range 150 {
		reachability.seeds = append(
			reachability.seeds,
			peerSeed(t, strconv.Itoa(index), "203.0.113.1", 8090),
		)
	}
	mux := muxWithHello(t, reachability, http.DefaultClient)

	resp := serveHello(
		t,
		mux,
		helloRequest("freeworld", unreachableCallerSeed(t), 150),
	)
	if got := len(resp.Seeds); got != 101 {
		t.Fatalf("Seeds = %d, want 101 (self + one hundred known)", got)
	}
}
