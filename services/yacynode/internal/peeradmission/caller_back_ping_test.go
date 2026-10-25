package peeradmission_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func TestCallerBackPingConfirmsValidQueryResponse(t *testing.T) {
	srv := backPingServer(true)
	defer srv.Close()

	reachability := &stubReachability{}
	mux := muxWithHello(t, reachability, srv.Client())

	resp := serveHello(
		t,
		mux,
		helloRequest("freeworld", reachableCallerSeed(t, srv.URL), 0),
	)

	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerSenior {
		t.Fatalf("YourType = %q, want senior for a confirming caller", yourType)
	}
}

func TestCallerBackPingRejectsErrorStatus(t *testing.T) {
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
		t.Fatalf("YourType = %q, want junior on error status", yourType)
	}
}

func TestCallerBackPingRejectsAReplyWithoutAWordCount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "<html lang=\"en\">\n")
	}))
	defer srv.Close()

	resp := serveHello(
		t,
		muxWithHello(t, &stubReachability{}, srv.Client()),
		helloRequest("freeworld", reachableCallerSeed(t, srv.URL), 0),
	)

	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerJunior {
		t.Fatalf("YourType = %q, want junior for a reply that is no word count", yourType)
	}
}

func TestCallerBackPingRejectsAReplyOfAnotherPeer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "response=-1\n")
	}))
	defer srv.Close()

	resp := serveHello(
		t,
		muxWithHello(t, &stubReachability{}, srv.Client()),
		helloRequest("freeworld", reachableCallerSeed(t, srv.URL), 0),
	)

	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerJunior {
		t.Fatalf("YourType = %q, want junior for a reply that rejects the addressed peer", yourType)
	}
}

func TestCallerBackPingReachesTheCallerThroughAnAdditionalHost(t *testing.T) {
	srv := backPingServer(true)
	defer srv.Close()
	caller := reachableCallerSeed(t, srv.URL)
	primary, err := yacymodel.ParseHost("127.0.0.2")
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}
	caller.AdditionalHosts = yacymodel.Some(
		[]yacymodel.Host{caller.PrimaryHost.OrElse(primary)},
	)
	caller.PrimaryHost = yacymodel.Some(primary)

	resp := serveHello(
		t,
		muxWithHello(t, &stubReachability{}, srv.Client()),
		helloRequest("freeworld", caller, 0),
	)

	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerSenior {
		t.Fatalf("YourType = %q, want senior for a caller behind its additional host", yourType)
	}
}

func TestCallerBackPingConfirmsTheCallerWithTheBaseURLThatAnswered(t *testing.T) {
	srv := backPingServer(true)
	defer srv.Close()
	reachability := &stubReachability{}
	caller := reachableCallerSeed(t, srv.URL)

	serveHello(t, muxWithHello(t, reachability, srv.Client()), helloRequest("freeworld", caller, 0))

	host, _ := caller.PrimaryHost.Get()
	port, _ := caller.Port.Get()
	want := peerwire.BaseURL{Scheme: peerwire.HTTP, Host: host, Port: port}
	if len(reachability.confirmedBaseURLs) != 1 || reachability.confirmedBaseURLs[0] != want {
		t.Fatalf("confirmed base URLs = %v, want %v", reachability.confirmedBaseURLs, want)
	}
}

func TestCallerBackPingStartsAtTheBaseURLThatAnsweredBefore(t *testing.T) {
	srv := backPingServer(true)
	defer srv.Close()
	var guard sync.Mutex
	var dialed []string
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, protocol, address string) (net.Conn, error) {
			guard.Lock()
			dialed = append(dialed, address)
			guard.Unlock()
			var dialer net.Dialer

			conn, err := dialer.DialContext(ctx, protocol, srv.Listener.Addr().String())
			if err != nil {
				return nil, fmt.Errorf("dial: %w", err)
			}

			return conn, nil
		},
		DisableKeepAlives: true,
	}}
	caller := peerSeed(t, callerHash, "192.0.2.1", 8090)
	additional, err := yacymodel.ParseHost("198.51.100.1")
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}
	caller.AdditionalHosts = yacymodel.Some([]yacymodel.Host{additional})
	reachability := &stubReachability{answeredBaseURLs: map[yacymodel.Hash]peerwire.BaseURL{
		caller.Hash: {Scheme: peerwire.HTTP, Host: additional, Port: 8090},
	}}

	serveHello(t, muxWithHello(t, reachability, client), helloRequest("freeworld", caller, 0))

	if len(dialed) == 0 || dialed[0] != "198.51.100.1:8090" {
		t.Fatalf("dialed = %v, want the base URL that answered before first", dialed)
	}
}

type expiredClock struct{ stoppedClock }

func (expiredClock) After(_ time.Duration, expire func()) (stop func()) {
	expire()

	return func() {}
}

type armingClock struct {
	stoppedClock
	armed chan struct{}
}

func (c armingClock) After(time.Duration, func()) (stop func()) {
	c.armed <- struct{}{}

	return func() {}
}

func busyBackPingServer(arrived chan<- struct{}, release <-chan struct{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		arrived <- struct{}{}
		<-release
		_, _ = io.WriteString(w, yacyproto.QueryResponse{Response: 3}.Encode().Encode())
	}))
}

func takeEveryBackPingSlot(
	t *testing.T,
	mux *http.ServeMux,
	caller yacymodel.Seed,
	arrived <-chan struct{},
) *sync.WaitGroup {
	t.Helper()

	var running sync.WaitGroup
	for range 16 {
		running.Go(func() { serveHello(t, mux, helloRequest("freeworld", caller, 0)) })
	}
	for range 16 {
		<-arrived
	}

	return &running
}

func TestCallerBackPingWaitsForAFreeSlot(t *testing.T) {
	arrived := make(chan struct{}, 17)
	release := make(chan struct{})
	srv := busyBackPingServer(arrived, release)
	defer srv.Close()
	clock := armingClock{armed: make(chan struct{}, 1)}
	mux := muxWithHelloAt(t, &stubReachability{}, srv.Client(), clock)
	caller := reachableCallerSeed(t, srv.URL)
	running := takeEveryBackPingSlot(t, mux, caller, arrived)

	replies := make(chan yacyproto.HelloResponse, 1)
	go func() { replies <- serveHello(t, mux, helloRequest("freeworld", caller, 0)) }()
	<-clock.armed
	close(release)
	resp := <-replies
	running.Wait()

	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerSenior {
		t.Fatalf("YourType = %q, want senior once a back-ping slot freed", yourType)
	}
}

func TestCallerBackPingSkipsACallerWhenNoSlotFreesInTime(t *testing.T) {
	arrived := make(chan struct{}, 16)
	release := make(chan struct{})
	srv := busyBackPingServer(arrived, release)
	defer srv.Close()
	mux := muxWithHelloAt(t, &stubReachability{}, srv.Client(), expiredClock{})
	caller := reachableCallerSeed(t, srv.URL)
	running := takeEveryBackPingSlot(t, mux, caller, arrived)

	resp := serveHello(t, mux, helloRequest("freeworld", caller, 0))
	close(release)
	running.Wait()

	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerJunior {
		t.Fatalf("YourType = %q, want junior when no back-ping slot frees in time", yourType)
	}
}
