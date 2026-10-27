package yggdrasilpeerchoice_test

import (
	"context"
	"crypto/ed25519"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilpeerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilpeerlist"
)

type listedPeers []yggdrasilpeerlist.Peer

func (l listedPeers) UpPeers(context.Context) []yggdrasilpeerlist.Peer {
	return l
}

type peeredNode struct {
	addedPeers []string
}

func (n *peeredNode) AddPeer(_ context.Context, peerURL *url.URL, _ ed25519.PublicKey) {
	n.addedPeers = append(n.addedPeers, peerURL.Host)
}

type connectTimes map[string]time.Duration

func (c connectTimes) ConnectTimeOf(_ context.Context, address string) (time.Duration, bool) {
	connectTime, connected := c[address]
	return connectTime, connected
}

func peerAt(host string, keyByte byte) yggdrasilpeerlist.Peer {
	key := make(ed25519.PublicKey, ed25519.PublicKeySize)
	key[0] = keyByte
	return yggdrasilpeerlist.Peer{URL: &url.URL{Scheme: "tls", Host: host}, Key: key}
}

func peersChosenFrom(t *testing.T, list listedPeers, times connectTimes) []string {
	t.Helper()
	node := &peeredNode{}
	yggdrasilpeerchoice.New(list, node, times).Choose(t.Context())
	return node.addedPeers
}

func TestChooseTakesTheThreePeersThatConnectFastest(t *testing.T) {
	list := listedPeers{
		peerAt("a:1", 1), peerAt("b:1", 2), peerAt("c:1", 3), peerAt("d:1", 4), peerAt("e:1", 5),
	}
	times := connectTimes{
		"a:1": 30 * time.Millisecond,
		"b:1": 10 * time.Millisecond,
		"d:1": 20 * time.Millisecond,
		"e:1": 40 * time.Millisecond,
	}

	chosenPeers := peersChosenFrom(t, list, times)

	if want := []string{"b:1", "d:1", "a:1"}; !slices.Equal(chosenPeers, want) {
		t.Errorf("node peered with %v, want %v", chosenPeers, want)
	}
}

func TestChooseTakesOnePeerForEachKey(t *testing.T) {
	list := listedPeers{peerAt("a:1", 1), peerAt("a:2", 1), peerAt("b:1", 2)}
	times := connectTimes{
		"a:1": 10 * time.Millisecond,
		"a:2": 20 * time.Millisecond,
		"b:1": 30 * time.Millisecond,
	}

	chosenPeers := peersChosenFrom(t, list, times)

	if want := []string{"a:1", "b:1"}; !slices.Equal(chosenPeers, want) {
		t.Errorf("node peered with %v, want %v", chosenPeers, want)
	}
}

func TestChooseTakesNoPeerWhenNoListedPeerConnects(t *testing.T) {
	chosenPeers := peersChosenFrom(t, listedPeers{peerAt("a:1", 1)}, connectTimes{})

	if len(chosenPeers) != 0 {
		t.Errorf("node peered with %v, want none", chosenPeers)
	}
}
