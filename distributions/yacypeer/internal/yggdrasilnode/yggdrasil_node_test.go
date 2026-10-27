package yggdrasilnode_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net"
	"net/netip"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilnode"
)

const (
	peeringDeadline  = 30 * time.Second
	peeringPollPause = 50 * time.Millisecond
	ipv6HeaderSize   = 40
	ipv6Version      = 0x60
	noNextHeader     = 59
	hopLimit         = 64
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

func listeningNode(t *testing.T) (*yggdrasilnode.Node, ed25519.PublicKey, *url.URL) {
	t.Helper()
	listenURL := &url.URL{Scheme: "unix", Path: filepath.Join(t.TempDir(), "ygg.sock")}
	node, key := startedNode(t, yggdrasilnode.Peering{ListenURLs: []*url.URL{listenURL}})
	return node, key, listenURL
}

func receivedPackets(node *yggdrasilnode.Node) <-chan struct{} {
	received := make(chan struct{}, 1)
	go func() {
		packet := make([]byte, node.MTU())
		for {
			if _, err := node.Read(packet); err != nil {
				return
			}
			select {
			case received <- struct{}{}:
			default:
			}
		}
	}()
	return received
}

func packetFrom(source, destination netip.Addr) []byte {
	packet := make([]byte, ipv6HeaderSize)
	packet[0] = ipv6Version
	packet[6] = noNextHeader
	packet[7] = hopLimit
	copy(packet[8:24], source.AsSlice())
	copy(packet[24:40], destination.AsSlice())
	return packet
}

func awaitPacketDelivery(t *testing.T, node, peerNode *yggdrasilnode.Node) {
	t.Helper()
	receivedPackets(node)
	received := receivedPackets(peerNode)
	packet := packetFrom(node.Address(), peerNode.Address())
	deadline := time.After(peeringDeadline)
	for {
		if _, err := node.Write(packet); err != nil {
			t.Fatalf("write a packet: %v", err)
		}
		select {
		case <-received:
			return
		case <-deadline:
			t.Fatalf("no packet reached %s", peerNode.Address())
		case <-time.After(peeringPollPause):
		}
	}
}

func relayTo(t *testing.T, peerURL *url.URL) (*url.URL, <-chan struct{}) {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for the relay: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	connectionEnded := make(chan struct{})
	go func() {
		defer close(connectionEnded)
		relayOnce(t.Context(), listener, peerURL)
	}()
	return &url.URL{Scheme: "tcp", Host: listener.Addr().String()}, connectionEnded
}

func relayOnce(ctx context.Context, listener net.Listener, peerURL *url.URL) {
	connection, err := listener.Accept()
	if err != nil {
		return
	}
	defer func() { _ = connection.Close() }()
	peerConnection, err := (&net.Dialer{}).DialContext(ctx, peerURL.Scheme, peerURL.Path)
	if err != nil {
		return
	}
	defer func() { _ = peerConnection.Close() }()
	go func() { _, _ = io.Copy(connection, peerConnection) }()
	_, _ = io.Copy(peerConnection, connection)
}

func TestAddPeerCarriesPacketsToThePeerOfTheKey(t *testing.T) {
	peerNode, peerKey, listenURL := listeningNode(t)
	node, _ := startedNode(t, yggdrasilnode.Peering{})

	node.AddPeer(t.Context(), listenURL, peerKey)

	awaitPacketDelivery(t, node, peerNode)
}

func TestRemovePeerEndsTheConnectionToThePeer(t *testing.T) {
	peerNode, peerKey, listenURL := listeningNode(t)
	relayURL, connectionEnded := relayTo(t, listenURL)
	node, _ := startedNode(t, yggdrasilnode.Peering{})
	node.AddPeer(t.Context(), relayURL, peerKey)
	awaitPacketDelivery(t, node, peerNode)

	node.RemovePeer(t.Context(), relayURL)

	select {
	case <-connectionEnded:
	case <-time.After(peeringDeadline):
		t.Fatal("the connection to the peer stays open")
	}
}
