package peeradmission_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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
