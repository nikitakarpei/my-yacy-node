package yggdrasilnode_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilnode"
)

const (
	peeringDeadline  = 30 * time.Second
	peeringPollPause = 50 * time.Millisecond
)

func startedNode(t *testing.T, peering yggdrasilnode.Peering) *yggdrasilnode.Node {
	t.Helper()
	node, _ := startedNodeWithKey(t, peering)
	return node
}

func startedNodeWithKey(
	t *testing.T,
	peering yggdrasilnode.Peering,
) (*yggdrasilnode.Node, ed25519.PublicKey) {
	t.Helper()
	publicKey, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("make a key: %v", err)
	}
	node, err := yggdrasilnode.Start(t.Context(), key, peering)
	if err != nil {
		t.Fatalf("start the node: %v", err)
	}
	t.Cleanup(node.Stop)
	return node, publicKey
}

func awaitPeering(t *testing.T, node *yggdrasilnode.Node, peerURL *url.URL, wantedUp bool) {
	t.Helper()
	deadline := time.Now().Add(peeringDeadline)
	for time.Now().Before(deadline) {
		if node.IsPeeredWith(peerURL) == wantedUp {
			return
		}
		time.Sleep(peeringPollPause)
	}
	t.Fatalf("peering with %s up = %t, want %t", peerURL, !wantedUp, wantedUp)
}

func TestAddPeerPeersTheNodeWithThePeerOfThePinnedKey(t *testing.T) {
	listenURL := &url.URL{Scheme: "unix", Path: filepath.Join(t.TempDir(), "ygg.sock")}
	_, peerKey := startedNodeWithKey(t, yggdrasilnode.Peering{ListenURLs: []*url.URL{listenURL}})
	node := startedNode(t, yggdrasilnode.Peering{})
	pinnedURL := *listenURL
	pinnedURL.RawQuery = url.Values{"key": {hex.EncodeToString(peerKey)}}.Encode()

	node.AddPeer(t.Context(), &pinnedURL)

	awaitPeering(t, node, &pinnedURL, true)
}

func TestRemovePeerEndsThePeeringWithThePeer(t *testing.T) {
	listenURL := &url.URL{Scheme: "unix", Path: filepath.Join(t.TempDir(), "ygg.sock")}
	startedNode(t, yggdrasilnode.Peering{ListenURLs: []*url.URL{listenURL}})
	node := startedNode(t, yggdrasilnode.Peering{PeerURLs: []*url.URL{listenURL}})
	awaitPeering(t, node, listenURL, true)

	node.RemovePeer(t.Context(), listenURL)

	awaitPeering(t, node, listenURL, false)
}
