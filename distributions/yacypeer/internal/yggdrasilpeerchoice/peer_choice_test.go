package yggdrasilpeerchoice_test

import (
	"context"
	"crypto/ed25519"
	"errors"
	"net/url"
	"slices"
	"sync"
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
}

func (n *peeredNode) AddPeer(_ context.Context, peerURL *url.URL, _ ed25519.PublicKey) {
	n.addedPeers = append(n.addedPeers, peerURL.Host)
}

func (n *peeredNode) RemovePeer(_ context.Context, peerURL *url.URL) {
	n.removedPeers = append(n.removedPeers, peerURL.Host)
}

type connectingPeers struct {
	mutex        sync.Mutex
	connectTimes map[string]time.Duration
	checkAnswers map[string][]bool
}

func connectingIn(connectTimes map[string]time.Duration) *connectingPeers {
	return &connectingPeers{connectTimes: connectTimes, checkAnswers: map[string][]bool{}}
}

func (c *connectingPeers) setCheckAnswers(address string, answers ...bool) {
	c.checkAnswers[address] = append([]bool{true}, answers...)
}

func (c *connectingPeers) ConnectTimeOf(_ context.Context, address string) (time.Duration, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	connectTime, listed := c.connectTimes[address]
	answers := c.checkAnswers[address]
	if len(answers) == 0 {
		return connectTime, listed
	}
	c.checkAnswers[address] = answers[1:]
	return connectTime, answers[0]
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

func peersConnectingInOrder(hosts ...string) (*listedPeers, *connectingPeers) {
	list := &listedPeers{}
	connectTimes := map[string]time.Duration{}
	for index, host := range hosts {
		list.peers = append(list.peers, peerAt(host, byte(index+1)))
		connectTimes[host] = time.Duration(index+1) * 10 * time.Millisecond
	}
	return list, connectingIn(connectTimes)
}

func choiceRun(
	t *testing.T,
	list *listedPeers,
	node *peeredNode,
	peers *connectingPeers,
	roundAmount int,
) {
	t.Helper()
	clock := &rounds{now: time.Unix(1_800_000_000, 0), left: roundAmount}
	choice := yggdrasilpeerchoice.New(list, node, peers, clock)
	if err := choice.Run(t.Context()); err != nil {
		t.Fatalf("run the choice: %v", err)
	}
}

func TestRunChoosesTheThreePeersThatConnectFastest(t *testing.T) {
	list := &listedPeers{peers: []yggdrasilpeerlist.Peer{
		peerAt("a:1", 1), peerAt("b:1", 2), peerAt("c:1", 3), peerAt("d:1", 4), peerAt("e:1", 5),
	}}
	node := &peeredNode{}
	peers := connectingIn(map[string]time.Duration{
		"a:1": 30 * time.Millisecond,
		"b:1": 10 * time.Millisecond,
		"d:1": 20 * time.Millisecond,
		"e:1": 40 * time.Millisecond,
	})

	choiceRun(t, list, node, peers, 1)

	if want := []string{"b:1", "d:1", "a:1"}; !slices.Equal(node.addedPeers, want) {
		t.Errorf("node peered with %v, want %v", node.addedPeers, want)
	}
}

func TestRunChoosesOnePeerForEachKey(t *testing.T) {
	list := &listedPeers{peers: []yggdrasilpeerlist.Peer{
		peerAt("a:1", 1), peerAt("a:2", 1), peerAt("b:1", 2),
	}}
	node := &peeredNode{}
	peers := connectingIn(map[string]time.Duration{
		"a:1": 10 * time.Millisecond,
		"a:2": 20 * time.Millisecond,
		"b:1": 30 * time.Millisecond,
	})

	choiceRun(t, list, node, peers, 1)

	if want := []string{"a:1", "b:1"}; !slices.Equal(node.addedPeers, want) {
		t.Errorf("node peered with %v, want %v", node.addedPeers, want)
	}
}

func TestRunReplacesAChosenPeerThatStaysDownWithTheNextFastest(t *testing.T) {
	list, peers := peersConnectingInOrder("a:1", "b:1", "c:1", "d:1")
	peers.setCheckAnswers("b:1", false, false)
	node := &peeredNode{}

	choiceRun(t, list, node, peers, 3)

	if !slices.Equal(node.removedPeers, []string{"b:1"}) ||
		!slices.Equal(node.addedPeers, []string{"a:1", "b:1", "c:1", "d:1"}) {
		t.Errorf("node peered with %v and dropped %v, want b:1 replaced by d:1",
			node.addedPeers, node.removedPeers)
	}
}

func TestRunKeepsAChosenPeerThatComesBackUp(t *testing.T) {
	list, peers := peersConnectingInOrder("a:1", "b:1", "c:1", "d:1")
	peers.setCheckAnswers("b:1", false, true, false)
	node := &peeredNode{}

	choiceRun(t, list, node, peers, 4)

	if len(node.removedPeers) != 0 {
		t.Errorf("node dropped %v, want b:1 kept", node.removedPeers)
	}
}

func TestRunReadsTheListAgainAnHourAfterItHadNoPeerLeft(t *testing.T) {
	list := &listedPeers{peers: []yggdrasilpeerlist.Peer{peerAt("a:1", 1)}}
	node := &peeredNode{}

	choiceRun(t, list, node, connectingIn(nil), 7)

	if list.reads != 2 {
		t.Errorf("list read %d times in an hour, want 2", list.reads)
	}
}
