package peeradmission_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peeradmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type helloRuntimeStatus struct{}

func (helloRuntimeStatus) Version(context.Context) string { return "1.0" }

func (helloRuntimeStatus) Uptime(context.Context) int { return 0 }

type stubReachability struct {
	seeds     []yacymodel.Seed
	refreshed []yacymodel.Hash
}

func (s *stubReachability) ReachablePeers(context.Context) []yacymodel.Seed {
	return s.seeds
}

func (s *stubReachability) ConfirmReachable(_ context.Context, peer yacymodel.Hash) {
	s.refreshed = append(s.refreshed, peer)
}

func muxWithHello(
	t *testing.T,
	reachability *stubReachability,
	client *http.Client,
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

	peeradmission.MountHello(router, localPeer(), selfStatus(t), reachability, client)

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
		yacyproto.ParseMessage(string(body)),
	)
	if err != nil {
		t.Fatalf("ParseHelloResponse: %v", err)
	}

	return resp
}

func backPingServer(reachable bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !reachable {
			http.Error(w, "boom", http.StatusInternalServerError)

			return
		}
		resp := yacyproto.QueryResponse{Response: 3}
		_, _ = io.WriteString(w, resp.Encode().Encode())
	}))
}

func TestHelloClassifiesReachableAddressedCallerAsSenior(t *testing.T) {
	srv := backPingServer(true)
	defer srv.Close()

	reachability := &stubReachability{
		seeds: []yacymodel.Seed{peerSeed(t, "trusted", "203.0.113.1", 8090)},
	}
	mux := muxWithHello(t, reachability, srv.Client())

	caller := reachableCallerSeed(t, srv.URL)
	resp := serveHello(t, mux, helloRequest("freeworld", caller, 0))

	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerSenior {
		t.Fatalf("YourType = %q, want senior", yourType)
	}
	if got := len(resp.Seeds); got != 2 {
		t.Fatalf("Seeds = %d, want 2 (self + trusted)", got)
	}
	if resp.Seeds[0].Hash != hashFor("self") {
		t.Fatalf("first seed = %q, want self", resp.Seeds[0].Hash)
	}
	if !slices.Equal(reachability.refreshed, []yacymodel.Hash{caller.Hash}) {
		t.Fatalf("refreshed = %v, want senior caller refreshed", reachability.refreshed)
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

func TestHelloOnForeignNetworkOmitsAdmission(t *testing.T) {
	reachability := &stubReachability{
		seeds: []yacymodel.Seed{peerSeed(t, "trusted", "203.0.113.1", 8090)},
	}
	mux := muxWithHello(t, reachability, http.DefaultClient)

	resp := serveHello(
		t,
		mux,
		helloRequest("otherworld", unreachableCallerSeed(t), 0),
	)
	if got := len(resp.Seeds); got != 1 {
		t.Fatalf("Seeds = %d, want 1 (self only)", got)
	}
	if resp.YourType.Present() {
		t.Fatalf("YourType = %v, want absent for foreign network", resp.YourType)
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

func TestHelloCountZeroReturnsAllKnownPeers(t *testing.T) {
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
	if got := len(resp.Seeds); got != 3 {
		t.Fatalf("Seeds = %d, want 3 (self + two known)", got)
	}
}
