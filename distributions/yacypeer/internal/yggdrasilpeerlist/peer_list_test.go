package yggdrasilpeerlist_test

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/yggdrasil-network/yggdrasil-go/src/core"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilpeerlist"
)

const (
	firstKey  = "2a038bfcc812ef82f0e4a936ccc065553b5beab0e566e2c48830eede873fdcc3"
	secondKey = "0000000000000000000000000000000000000000000000000000000000000001"
)

func listServing(t *testing.T, status int, body string) *yggdrasilpeerlist.List {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	listURL, err := url.Parse(server.URL + "/publicnodes.json")
	if err != nil {
		t.Fatalf("parse the list URL: %v", err)
	}
	return yggdrasilpeerlist.New(listURL)
}

func listedEntry(peerURL string, key string, up bool, protocolMinorVersion uint16) string {
	return fmt.Sprintf(`%q: {"up": %t, "key": %q, "proto_minor": %d}`,
		peerURL, up, key, protocolMinorVersion)
}

func upEntry(peerURL string, key string) string {
	return listedEntry(peerURL, key, true, core.ProtocolVersionMinor)
}

func peerURLsOf(peers []yggdrasilpeerlist.Peer) []string {
	peerURLs := make([]string, 0, len(peers))
	for _, peer := range peers {
		peerURLs = append(peerURLs, peer.URL.String())
	}
	return peerURLs
}

func TestUpPeersGivesEachPeerThatIsUpWithItsListedKey(t *testing.T) {
	list := listServing(t, http.StatusOK, `{
		"armenia.md": {`+upEntry("tls://37.186.113.100:1515?sni=peer.example", firstKey)+`},
		"germany.md": {`+upEntry("tcp://198.51.100.7:1514", secondKey)+`}
	}`)

	upPeers := list.UpPeers(t.Context())

	wantURLs := []string{"tcp://198.51.100.7:1514", "tls://37.186.113.100:1515?sni=peer.example"}
	if got := peerURLsOf(upPeers); !slices.Equal(got, wantURLs) {
		t.Fatalf("up peers %v, want %v", got, wantURLs)
	}
	if got := hex.EncodeToString(upPeers[1].Key); got != firstKey {
		t.Errorf("key of %s is %s, want %s", upPeers[1].URL, got, firstKey)
	}
}

func TestUpPeersLeavesOutAPeerTheNodeCannotUse(t *testing.T) {
	const peerURL = "tcp://198.51.100.7:1514"
	for name, listed := range map[string]string{
		"down":           listedEntry(peerURL, firstKey, false, core.ProtocolVersionMinor),
		"older protocol": listedEntry(peerURL, firstKey, true, core.ProtocolVersionMinor-1),
		"quic":           upEntry("quic://198.51.100.7:1515", firstKey),
		"short key":      upEntry(peerURL, "2a03"),
		"key not hex":    upEntry(peerURL, strings.Repeat("zz", 32)),
		"unreadable URL": upEntry("tcp://198.51.100.7:port", firstKey),
	} {
		t.Run(name, func(t *testing.T) {
			list := listServing(t, http.StatusOK, `{"germany.md": {`+listed+`}}`)

			if upPeers := list.UpPeers(t.Context()); len(upPeers) != 0 {
				t.Errorf("up peers %v, want none", peerURLsOf(upPeers))
			}
		})
	}
}

func TestUpPeersHasNoPeerWhenTheListIsNotRead(t *testing.T) {
	for name, list := range map[string]*yggdrasilpeerlist.List{
		"refused":      listServing(t, http.StatusServiceUnavailable, ""),
		"not the list": listServing(t, http.StatusOK, "<html></html>"),
	} {
		t.Run(name, func(t *testing.T) {
			if upPeers := list.UpPeers(t.Context()); len(upPeers) != 0 {
				t.Errorf("up peers %v, want none", peerURLsOf(upPeers))
			}
		})
	}
}
