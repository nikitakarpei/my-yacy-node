package yggdrasilpeerlist_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

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

func peerURLsOf(peers []yggdrasilpeerlist.Peer) []string {
	peerURLs := make([]string, 0, len(peers))
	for _, peer := range peers {
		peerURLs = append(peerURLs, peer.URL.String())
	}
	return peerURLs
}

func TestUpPeersPinsEachPeerThatIsUpToItsListedKey(t *testing.T) {
	list := listServing(t, http.StatusOK, `{
		"armenia.md": {
			"tls://37.186.113.100:1515": {"up": true, "key": "`+firstKey+`", "proto_minor": 5}
		},
		"germany.md": {
			"tcp://198.51.100.7:1514": {"up": true, "key": "`+secondKey+`", "proto_minor": 5}
		}
	}`)

	upPeers := list.UpPeers(t.Context())

	want := []string{
		"tcp://198.51.100.7:1514?key=" + secondKey,
		"tls://37.186.113.100:1515?key=" + firstKey,
	}
	if got := peerURLsOf(upPeers); !slices.Equal(got, want) {
		t.Errorf("up peers %v, want %v", got, want)
	}
	if len(upPeers) == 2 && upPeers[1].Key[0] != 0x2a {
		t.Errorf("key of %s is %x, want the listed key", upPeers[1].URL, upPeers[1].Key)
	}
}

func TestUpPeersKeepsTheOtherParametersOfAPeerURL(t *testing.T) {
	list := listServing(t, http.StatusOK, `{"france.md": {
		"tls://peer.example:443?sni=peer.example": {"up": true, "key": "`+firstKey+`", "proto_minor": 5}
	}}`)

	got := peerURLsOf(list.UpPeers(t.Context()))

	want := []string{"tls://peer.example:443?key=" + firstKey + "&sni=peer.example"}
	if !slices.Equal(got, want) {
		t.Errorf("up peers %v, want %v", got, want)
	}
}

func TestUpPeersLeavesOutAPeerTheNodeCannotUse(t *testing.T) {
	for name, listed := range map[string]string{
		"down":           `"tcp://198.51.100.7:1514": {"up": false, "key": "` + firstKey + `", "proto_minor": 5}`,
		"older protocol": `"tcp://198.51.100.7:1514": {"up": true, "key": "` + firstKey + `", "proto_minor": 4}`,
		"quic":           `"quic://198.51.100.7:1515": {"up": true, "key": "` + firstKey + `", "proto_minor": 5}`,
		"short key":      `"tcp://198.51.100.7:1514": {"up": true, "key": "2a03", "proto_minor": 5}`,
		"key not hex":    `"tcp://198.51.100.7:1514": {"up": true, "key": "` + strings.Repeat("zz", 32) + `", "proto_minor": 5}`,
		"unreadable URL": `"tcp://198.51.100.7:port": {"up": true, "key": "` + firstKey + `", "proto_minor": 5}`,
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
