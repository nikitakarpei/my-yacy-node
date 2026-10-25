package peeradmission_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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
