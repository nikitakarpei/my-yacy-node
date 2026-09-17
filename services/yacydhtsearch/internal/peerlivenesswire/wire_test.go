package peerlivenesswire_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenesswire"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	networkName            = "freeworld"
	rwiCountAnswer         = "version=1.83\nuptime=1200\nresponse=42\n"
	rwiCountRejectedAnswer = "version=1.83\nuptime=1200\nresponse=-1\n"
	unparseableAnswer      = "<html><body>not a peer</body></html>"
)

func peerHash(t *testing.T, symbol byte) yacymodel.Hash {
	t.Helper()

	hash, err := yacymodel.ParseHash(string([]byte{
		symbol, symbol, symbol, symbol, symbol, symbol,
		symbol, symbol, symbol, symbol, symbol, symbol,
	}))
	if err != nil {
		t.Fatalf("ParseHash: %v", err)
	}

	return hash
}

func addressAnswering(t *testing.T, status int, answer string) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(status)
			_, _ = writer.Write([]byte(answer))
		},
	))
	t.Cleanup(server.Close)

	return server.URL
}

func addressOfPeer(t *testing.T, peer yacymodel.Hash) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Query().Get(yacyproto.FieldYouAre) != peer.String() {
				_, _ = writer.Write([]byte(rwiCountRejectedAnswer))

				return
			}
			_, _ = writer.Write([]byte(rwiCountAnswer))
		},
	))
	t.Cleanup(server.Close)

	return server.URL
}

func wire() peerlivenesswire.Wire {
	return peerlivenesswire.New(
		http.DefaultClient, networkName, peerlivenesswire.PeerLivenessObservers{},
	)
}

func TestAPeerThatAnswersTheProbeWithItsPeerCountIsAlive(t *testing.T) {
	t.Parallel()

	peer := peerHash(t, 'A')
	if !wire().Alive(t.Context(), peer, addressOfPeer(t, peer)) {
		t.Fatal("Alive = false for a peer that answers the probe that names it")
	}
}

func TestAPeerAskedUnderTheHashOfAnotherPeerIsNotAlive(t *testing.T) {
	t.Parallel()

	if wire().Alive(t.Context(), peerHash(t, 'B'), addressOfPeer(t, peerHash(t, 'A'))) {
		t.Fatal("Alive = true for an address that holds another peer")
	}
}

func TestAnAddressThatAnswersSomethingOtherThanAPeerCountIsNotAlive(t *testing.T) {
	t.Parallel()

	address := addressAnswering(t, http.StatusOK, unparseableAnswer)
	if wire().Alive(t.Context(), peerHash(t, 'A'), address) {
		t.Fatal("Alive = true for an address that answered no peer count")
	}
}

func TestAPeerThatRejectsThePeerCountQueryIsNotAlive(t *testing.T) {
	t.Parallel()

	address := addressAnswering(t, http.StatusOK, rwiCountRejectedAnswer)
	if wire().Alive(t.Context(), peerHash(t, 'A'), address) {
		t.Fatal("Alive = true for a peer that rejected the peer count query")
	}
}

func TestAnAddressThatRefusesTheProbeIsNotAlive(t *testing.T) {
	t.Parallel()

	address := addressAnswering(t, http.StatusForbidden, "")
	if wire().Alive(t.Context(), peerHash(t, 'A'), address) {
		t.Fatal("Alive = true for an address that refused the probe")
	}
}

func TestAnAddressNothingListensOnIsNotAlive(t *testing.T) {
	t.Parallel()

	if wire().Alive(t.Context(), peerHash(t, 'A'), "http://127.0.0.1:1") {
		t.Fatal("Alive = true for an address nothing listens on")
	}
}
