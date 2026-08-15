package bootstrap_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/bootstrap"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func mustPeerName(t *testing.T, name string) yacymodel.PeerName {
	t.Helper()

	peerName, err := yacymodel.ParsePeerName(name)
	if err != nil {
		t.Fatalf("parse peer name: %v", err)
	}

	return peerName
}

func mustHash(t *testing.T, raw string) yacymodel.Hash {
	t.Helper()
	hash, err := yacymodel.ParseHash(raw)
	if err != nil {
		t.Fatalf("parse hash: %v", err)
	}

	return hash
}

func seedlistLine(t *testing.T, hash, ip string) string {
	t.Helper()

	host, err := yacymodel.ParseHost(ip)
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}
	seed := yacymodel.Seed{
		Hash:           mustHash(t, hash),
		Name:           mustPeerName(t, "peer-"+hash),
		PeerType:       yacymodel.PeerSenior,
		PrimaryAddress: yacymodel.Some(host),
		Port:           yacymodel.Some(yacymodel.Port(8090)),
	}

	return yacyproto.EncodeSeed(seed)
}

func TestSeedlistFetcherDecodesLines(t *testing.T) {
	body := strings.Join([]string{
		seedlistLine(t, "AAAAAAAAAAAA", "203.0.113.1"),
		"",
		"!!! not a seed line",
		seedlistLine(t, "BBBBBBBBBBBB", "203.0.113.2"),
	}, "\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	source := bootstrap.New(server.Client(), []string{server.URL})
	seeds := source.Fetch(context.Background())
	if len(seeds) != 2 {
		t.Fatalf("got %d seeds, want 2 (bad line skipped)", len(seeds))
	}
}

func TestSeedlistFetcherRejectsNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer server.Close()

	source := bootstrap.New(server.Client(), []string{server.URL})
	seeds := source.Fetch(context.Background())
	if len(seeds) != 0 {
		t.Fatalf("got %d seeds, want 0 (non-200 response rejected)", len(seeds))
	}
}
