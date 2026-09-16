package peerlivenesswire_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenesswire"
)

const (
	networkName            = "freeworld"
	rwiCountAnswer         = "version=1.83\nuptime=1200\nresponse=42\n"
	rwiCountRejectedAnswer = "version=1.83\nuptime=1200\nresponse=-1\n"
	unparseableAnswer      = "<html><body>not a peer</body></html>"
)

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

func wire() peerlivenesswire.Wire {
	return peerlivenesswire.New(
		http.DefaultClient, networkName, peerlivenesswire.PeerLivenessObservers{},
	)
}

func TestAPeerThatAnswersTheProbeWithItsPeerCountIsAlive(t *testing.T) {
	t.Parallel()

	if !wire().Alive(t.Context(), addressAnswering(t, http.StatusOK, rwiCountAnswer)) {
		t.Fatal("Alive = false for a peer that answered the probe with its peer count")
	}
}

func TestAnAddressThatAnswersSomethingOtherThanAPeerCountIsNotAlive(t *testing.T) {
	t.Parallel()

	if wire().Alive(t.Context(), addressAnswering(t, http.StatusOK, unparseableAnswer)) {
		t.Fatal("Alive = true for an address that answered no peer count")
	}
}

func TestAPeerThatRejectsThePeerCountQueryIsNotAlive(t *testing.T) {
	t.Parallel()

	if wire().Alive(t.Context(), addressAnswering(t, http.StatusOK, rwiCountRejectedAnswer)) {
		t.Fatal("Alive = true for a peer that rejected the peer count query")
	}
}

func TestAnAddressThatRefusesTheProbeIsNotAlive(t *testing.T) {
	t.Parallel()

	if wire().Alive(t.Context(), addressAnswering(t, http.StatusForbidden, "")) {
		t.Fatal("Alive = true for an address that refused the probe")
	}
}

func TestAnAddressNothingListensOnIsNotAlive(t *testing.T) {
	t.Parallel()

	if wire().Alive(t.Context(), "http://127.0.0.1:1") {
		t.Fatal("Alive = true for an address nothing listens on")
	}
}
