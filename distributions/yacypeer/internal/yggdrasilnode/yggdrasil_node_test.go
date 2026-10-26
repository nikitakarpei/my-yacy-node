package yggdrasilnode_test

import (
	"crypto/ed25519"
	"crypto/rand"
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

func startedNode(
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

func awaitPeering(t *testing.T, node *yggdrasilnode.Node, key ed25519.PublicKey, wantedUp bool) {
	t.Helper()
	deadline := time.Now().Add(peeringDeadline)
	for time.Now().Before(deadline) {
		if node.IsPeeredWith(key) == wantedUp {
			return
		}
		time.Sleep(peeringPollPause)
	}
	t.Fatalf("peering with %x up = %t, want %t", key, !wantedUp, wantedUp)
}

func TestAddPeerPeersTheNodeWithThePeerOfTheKey(t *testing.T) {
	listenURL := &url.URL{Scheme: "unix", Path: filepath.Join(t.TempDir(), "ygg.sock")}
	_, peerKey := startedNode(t, yggdrasilnode.Peering{ListenURLs: []*url.URL{listenURL}})
	node, _ := startedNode(t, yggdrasilnode.Peering{})

	node.AddPeer(t.Context(), listenURL, peerKey)

	awaitPeering(t, node, peerKey, true)
}

func TestRemovePeerEndsThePeeringWithThePeer(t *testing.T) {
	listenURL := &url.URL{Scheme: "unix", Path: filepath.Join(t.TempDir(), "ygg.sock")}
	_, peerKey := startedNode(t, yggdrasilnode.Peering{ListenURLs: []*url.URL{listenURL}})
	node, _ := startedNode(t, yggdrasilnode.Peering{})
	node.AddPeer(t.Context(), listenURL, peerKey)
	awaitPeering(t, node, peerKey, true)

	node.RemovePeer(t.Context(), listenURL)

	awaitPeering(t, node, peerKey, false)
}
