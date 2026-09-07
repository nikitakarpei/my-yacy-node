package peerlivenesswire_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenesswire"
)

func peerAnswering(t *testing.T, status int) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(status) },
	))
	t.Cleanup(server.Close)

	return server.URL
}

func TestAPeerThatAnswersTheProbeIsAlive(t *testing.T) {
	t.Parallel()

	wire := peerlivenesswire.New(http.DefaultClient, "freeworld")

	if !wire.Alive(t.Context(), peerAnswering(t, http.StatusOK)) {
		t.Fatal("Alive = false for a peer that answered the probe")
	}
}

func TestAPeerOfAnotherNetworkIsNotAlive(t *testing.T) {
	t.Parallel()

	wire := peerlivenesswire.New(http.DefaultClient, "freeworld")

	if wire.Alive(t.Context(), peerAnswering(t, http.StatusForbidden)) {
		t.Fatal("Alive = true for a peer that refused the probe")
	}
}

func TestAnAddressNothingListensOnIsNotAlive(t *testing.T) {
	t.Parallel()

	wire := peerlivenesswire.New(http.DefaultClient, "freeworld")

	if wire.Alive(t.Context(), "http://127.0.0.1:1") {
		t.Fatal("Alive = true for an address nothing listens on")
	}
}
