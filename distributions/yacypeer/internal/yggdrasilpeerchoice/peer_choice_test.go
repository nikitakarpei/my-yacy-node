package yggdrasilpeerchoice_test

import (
	"context"
	"crypto/ed25519"
	"errors"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilpeerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilpeerlist"
)

var errChoiceStops = errors.New("choice stops")

type listedPeers struct {
	peers []yggdrasilpeerlist.Peer
	reads int
}

func (l *listedPeers) UpPeers(context.Context) []yggdrasilpeerlist.Peer {
	l.reads++
	return l.peers
}

type peeredNode struct {
	addedPeers   []string
	removedPeers []string
	upAnswers    map[byte][]bool
}

func (n *peeredNode) AddPeer(_ context.Context, peerURL *url.URL, _ ed25519.PublicKey) {
	n.addedPeers = append(n.addedPeers, peerURL.Host)
}

func (n *peeredNode) RemovePeer(_ context.Context, peerURL *url.URL) {
	n.removedPeers = append(n.removedPeers, peerURL.Host)
}

func (n *peeredNode) IsPeeredWith(key ed25519.PublicKey) bool {
	answers := n.upAnswers[key[0]]
	if len(answers) == 0 {
		return true
	}
	n.upAnswers[key[0]] = answers[1:]
	return answers[0]
}

type connectTimes map[string]time.Duration

func (c connectTimes) ConnectTimeOf(_ context.Context, address string) (time.Duration, bool) {
	connectTime, connected := c[address]
	return connectTime, connected
}

type rounds struct {
	now  time.Time
	left int
}

func (r *rounds) Now() time.Time {
	return r.now
}

func (r *rounds) Sleep(_ context.Context, d time.Duration) error {
	r.now = r.now.Add(d)
	r.left--
	if r.left <= 0 {
		return errChoiceStops
	}
	return nil
}

func peerAt(host string, keyByte byte) yggdrasilpeerlist.Peer {
	key := make(ed25519.PublicKey, ed25519.PublicKeySize)
	key[0] = keyByte
	return yggdrasilpeerlist.Peer{URL: &url.URL{Scheme: "tls", Host: host}, Key: key}
}

func peersConnectingInOrder(hosts ...string) (*listedPeers, connectTimes) {
	list := &listedPeers{}
	times := connectTimes{}
	for index, host := range hosts {
		list.peers = append(list.peers, peerAt(host, byte(index+1)))
		times[host] = time.Duration(index+1) * 10 * time.Millisecond
	}
	return list, times
}

func choiceRun(
	t *testing.T,
	list *listedPeers,
	node *peeredNode,
	times connectTimes,
	roundAmount int,
) {
	t.Helper()
	clock := &rounds{now: time.Unix(1_800_000_000, 0), left: roundAmount}
	choice := yggdrasilpeerchoice.New(list, node, times, clock)
	if err := choice.Run(t.Context()); err != nil {
		t.Fatalf("run the choice: %v", err)
	}
}

func TestRunChoosesTheThreePeersThatConnectFastest(t *testing.T) {
	list := &listedPeers{peers: []yggdrasilpeerlist.Peer{
		peerAt("a:1", 1), peerAt("b:1", 2), peerAt("c:1", 3), peerAt("d:1", 4), peerAt("e:1", 5),
	}}
	node := &peeredNode{}
	times := connectTimes{
		"a:1": 30 * time.Millisecond,
		"b:1": 10 * time.Millisecond,
		"d:1": 20 * time.Millisecond,
		"e:1": 40 * time.Millisecond,
	}

	choiceRun(t, list, node, times, 1)

	if want := []string{"b:1", "d:1", "a:1"}; !slices.Equal(node.addedPeers, want) {
		t.Errorf("node peered with %v, want %v", node.addedPeers, want)
	}
}

func TestRunChoosesOnePeerForEachKey(t *testing.T) {
	list := &listedPeers{peers: []yggdrasilpeerlist.Peer{
		peerAt("a:1", 1), peerAt("a:2", 1), peerAt("b:1", 2),
	}}
	node := &peeredNode{}
	times := connectTimes{
		"a:1": 10 * time.Millisecond,
		"a:2": 20 * time.Millisecond,
		"b:1": 30 * time.Millisecond,
	}

	choiceRun(t, list, node, times, 1)

	if want := []string{"a:1", "b:1"}; !slices.Equal(node.addedPeers, want) {
		t.Errorf("node peered with %v, want %v", node.addedPeers, want)
	}
}

func TestRunReplacesAChosenPeerThatStaysDownWithTheNextFastest(t *testing.T) {
	list, times := peersConnectingInOrder("a:1", "b:1", "c:1", "d:1")
	node := &peeredNode{upAnswers: map[byte][]bool{2: {false, false}}}

	choiceRun(t, list, node, times, 3)

	if !slices.Equal(node.removedPeers, []string{"b:1"}) ||
		!slices.Equal(node.addedPeers, []string{"a:1", "b:1", "c:1", "d:1"}) {
		t.Errorf("node peered with %v and dropped %v, want b:1 replaced by d:1",
			node.addedPeers, node.removedPeers)
	}
}

func TestRunKeepsAChosenPeerThatComesBackUp(t *testing.T) {
	list, times := peersConnectingInOrder("a:1", "b:1", "c:1", "d:1")
	node := &peeredNode{upAnswers: map[byte][]bool{2: {false, true, false}}}

	choiceRun(t, list, node, times, 4)

	if len(node.removedPeers) != 0 {
		t.Errorf("node dropped %v, want b:1 kept", node.removedPeers)
	}
}

func TestRunReadsTheListAgainAnHourAfterItHadNoPeerLeft(t *testing.T) {
	list := &listedPeers{peers: []yggdrasilpeerlist.Peer{peerAt("a:1", 1)}}
	node := &peeredNode{}

	choiceRun(t, list, node, connectTimes{}, 62)

	if list.reads != 2 {
		t.Errorf("list read %d times in 61 minutes, want 2", list.reads)
	}
}
