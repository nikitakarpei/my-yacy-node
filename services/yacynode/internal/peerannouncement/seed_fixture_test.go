package peerannouncement_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	hashFiller  = "AAAAAAAAAAAA"
	networkName = "freeworld"
)

func hashFor(base string) yacymodel.Hash {
	padded := base + hashFiller
	hash, err := yacymodel.ParseHash(padded[:yacymodel.HashLength])
	if err != nil {
		panic(err)
	}

	return hash
}

func peerSeedAt(t testing.TB, hash, host string, port int) yacymodel.Seed {
	t.Helper()

	address, err := yacymodel.ParseHost(host)
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}
	name, err := yacymodel.ParsePeerName(hashFor(hash).String())
	if err != nil {
		t.Fatalf("parse peer name: %v", err)
	}

	return yacymodel.Seed{
		Hash:           hashFor(hash),
		Name:           name,
		PeerType:       yacymodel.PeerSenior,
		PrimaryAddress: yacymodel.Some(address),
		Port:           yacymodel.Some(yacymodel.Port(port)),
	}
}

func hostPortOf(t testing.TB, rawURL string) (string, int) {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse peer url: %v", err)
	}
	host, portText, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("split peer host: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parse peer port: %v", err)
	}

	return host, port
}

func answeringPeerSeed(t testing.TB) yacymodel.Seed {
	t.Helper()

	return peerSeedAt(t, "answering", "203.0.113.5", 8090)
}

type peerAnswer func(yacymodel.Seed) http.HandlerFunc

type stubPeer struct {
	seed   yacymodel.Seed
	answer http.HandlerFunc
	greets atomic.Int64
}

func newStubPeer(t testing.TB, hash string, answer peerAnswer) *stubPeer {
	t.Helper()

	peer := &stubPeer{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer.greets.Add(1)
		if r.URL.Path != yacyproto.PathHello {
			t.Errorf("path = %q, want %q", r.URL.Path, yacyproto.PathHello)
		}
		peer.answer(w, r)
	}))
	t.Cleanup(server.Close)
	host, port := hostPortOf(t, server.URL)
	peer.seed = peerSeedAt(t, hash, host, port)
	peer.answer = answer(peer.seed)

	return peer
}

func (p *stubPeer) greetCount() int {
	return int(p.greets.Load())
}

func helloAnswer(response yacyproto.HelloResponse) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(response.Encode().Encode()))
	}
}

func seniorAnswer(t testing.TB, gossiped ...yacymodel.Seed) peerAnswer {
	t.Helper()

	return func(respondingSeed yacymodel.Seed) http.HandlerFunc {
		return helloAnswer(yacyproto.HelloResponse{
			YourIP:   "203.0.113.9",
			YourType: yacymodel.Some(yacymodel.PeerSenior),
			Seeds:    append([]yacymodel.Seed{respondingSeed}, gossiped...),
		})
	}
}

func virginAnswer(t testing.TB, gossiped ...yacymodel.Seed) peerAnswer {
	t.Helper()

	return func(respondingSeed yacymodel.Seed) http.HandlerFunc {
		return helloAnswer(yacyproto.HelloResponse{
			YourIP:   "203.0.113.9",
			YourType: yacymodel.Some(yacymodel.PeerVirgin),
			Seeds:    append([]yacymodel.Seed{respondingSeed}, gossiped...),
		})
	}
}

func untypedAnswer(t testing.TB, gossiped ...yacymodel.Seed) peerAnswer {
	t.Helper()

	return func(respondingSeed yacymodel.Seed) http.HandlerFunc {
		return helloAnswer(yacyproto.HelloResponse{
			YourIP: "203.0.113.9",
			Seeds:  append([]yacymodel.Seed{respondingSeed}, gossiped...),
		})
	}
}

func answerFromReplacementPeer(t testing.TB) peerAnswer {
	t.Helper()

	return func(yacymodel.Seed) http.HandlerFunc {
		return helloAnswer(yacyproto.HelloResponse{
			YourIP:   "203.0.113.9",
			YourType: yacymodel.Some(yacymodel.PeerSenior),
			Seeds:    []yacymodel.Seed{answeringPeerSeed(t)},
		})
	}
}

func unavailableAnswer() peerAnswer {
	return func(yacymodel.Seed) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "nope", http.StatusServiceUnavailable)
		}
	}
}
