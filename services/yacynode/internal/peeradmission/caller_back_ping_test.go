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

func TestCallerBackPingReachesTheCallerThroughAnAdditionalAddress(t *testing.T) {
	srv := backPingServer(true)
	defer srv.Close()
	caller := reachableCallerSeed(t, srv.URL)
	primary, err := yacymodel.ParseHost("127.0.0.2")
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}
	caller.AdditionalAddresses = yacymodel.Some(
		[]yacymodel.Host{caller.PrimaryAddress.OrElse(primary)},
	)
	caller.PrimaryAddress = yacymodel.Some(primary)

	resp := serveHello(
		t,
		muxWithHello(t, &stubReachability{}, srv.Client()),
		helloRequest("freeworld", caller, 0),
	)

	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerSenior {
		t.Fatalf("YourType = %q, want senior for a caller behind its additional address", yourType)
	}
}

func TestCallerBackPingConfirmsTheCallerWithTheOriginThatAnswered(t *testing.T) {
	srv := backPingServer(true)
	defer srv.Close()
	reachability := &stubReachability{}
	caller := reachableCallerSeed(t, srv.URL)

	serveHello(t, muxWithHello(t, reachability, srv.Client()), helloRequest("freeworld", caller, 0))

	host, _ := caller.PrimaryAddress.Get()
	port, _ := caller.Port.Get()
	want := peerwire.Origin{Scheme: peerwire.HTTP, Host: host, Port: port}
	if len(reachability.confirmedOrigins) != 1 || reachability.confirmedOrigins[0] != want {
		t.Fatalf("confirmed origins = %v, want %v", reachability.confirmedOrigins, want)
	}
}

func TestCallerBackPingStartsAtTheOriginThatAnsweredBefore(t *testing.T) {
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
	caller.AdditionalAddresses = yacymodel.Some([]yacymodel.Host{additional})
	reachability := &stubReachability{answeredOrigins: map[yacymodel.Hash]peerwire.Origin{
		caller.Hash: {Scheme: peerwire.HTTP, Host: additional, Port: 8090},
	}}

	serveHello(t, muxWithHello(t, reachability, client), helloRequest("freeworld", caller, 0))

	if len(dialed) == 0 || dialed[0] != "198.51.100.1:8090" {
		t.Fatalf("dialed = %v, want the origin that answered before first", dialed)
	}
}

func TestCallerBackPingSkipsACallerWhileSixteenBackPingsRun(t *testing.T) {
	arrived := make(chan struct{}, 16)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		arrived <- struct{}{}
		<-release
		_, _ = io.WriteString(w, yacyproto.QueryResponse{Response: 3}.Encode().Encode())
	}))
	defer srv.Close()
	mux := muxWithHello(t, &stubReachability{}, srv.Client())
	caller := reachableCallerSeed(t, srv.URL)

	var running sync.WaitGroup
	for range 16 {
		running.Go(func() { serveHello(t, mux, helloRequest("freeworld", caller, 0)) })
	}
	for range 16 {
		<-arrived
	}
	resp := serveHello(t, mux, helloRequest("freeworld", caller, 0))
	close(release)
	running.Wait()

	if yourType, _ := resp.YourType.Get(); yourType != yacymodel.PeerJunior {
		t.Fatalf("YourType = %q, want junior while every back-ping slot is taken", yourType)
	}
}
