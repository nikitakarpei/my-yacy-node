// Package yggdrasilpeerchoice chooses the public peers of the Yggdrasil node. It peers the node
// with the public peers that the peer list states are up and that connect fastest from here,
// each with its own key. It looks at the chosen peers again after a pause, replaces a chosen peer
// that stays down with the next fastest one, and reads the list again when it has no peer left.
package yggdrasilpeerchoice

import (
	"context"
	"log/slog"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilpeerlist"
)

const (
	wantedPeers     = 3
	roundPause      = time.Minute
	downRoundsLimit = 2
	listReadPause   = time.Hour
)

type PeerList interface {
	UpPeers(ctx context.Context) []yggdrasilpeerlist.Peer
}

type Node interface {
	AddPeer(ctx context.Context, peerURL *url.URL)
	RemovePeer(ctx context.Context, peerURL *url.URL)
	IsPeeredWith(peerURL *url.URL) bool
}

type ConnectTimer interface {
	ConnectTimeOf(ctx context.Context, address string) (time.Duration, bool)
}

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type chosenPeer struct {
	peer       yggdrasilpeerlist.Peer
	roundsDown int
}

type Choice struct {
	list         PeerList
	node         Node
	connectTimer ConnectTimer
	clock        Clock
	chosenPeers  []chosenPeer
	fastPeers    []rankedPeer
	listReadAt   time.Time
}

func New(list PeerList, node Node, connectTimer ConnectTimer, clock Clock) *Choice {
	return &Choice{list: list, node: node, connectTimer: connectTimer, clock: clock}
}

func (c *Choice) Run(ctx context.Context) error {
	for {
		c.dropPeersThatStayDown(ctx)
		c.chooseWantedPeers(ctx)
		if err := c.clock.Sleep(ctx, roundPause); err != nil {
			return nil //nolint:nilerr // a clock only stops sleeping early when the choice is to stop
		}
	}
}

func (c *Choice) dropPeersThatStayDown(ctx context.Context) {
	keptPeers := c.chosenPeers[:0]
	for _, chosen := range c.chosenPeers {
		chosen.roundsDown = roundsDownAfter(chosen, c.node.IsPeeredWith(chosen.peer.URL))
		if chosen.roundsDown < downRoundsLimit {
			keptPeers = append(keptPeers, chosen)
			continue
		}
		c.node.RemovePeer(ctx, chosen.peer.URL)
		slog.WarnContext(ctx, "public Yggdrasil peer dropped: it stays down",
			slog.String("peer", chosen.peer.URL.Redacted()),
		)
	}
	c.chosenPeers = keptPeers
}

func roundsDownAfter(chosen chosenPeer, isUp bool) int {
	if isUp {
		return 0
	}
	return chosen.roundsDown + 1
}

func (c *Choice) chooseWantedPeers(ctx context.Context) {
	for len(c.chosenPeers) < wantedPeers {
		next, found := c.nextFastPeer()
		switch {
		case found:
			c.choose(ctx, next)
		case c.mayReadList():
			c.readList(ctx)
		default:
			return
		}
	}
}

func (c *Choice) nextFastPeer() (rankedPeer, bool) {
	for len(c.fastPeers) > 0 {
		next := c.fastPeers[0]
		c.fastPeers = c.fastPeers[1:]
		if !c.hasChosenKeyOf(next.peer) {
			return next, true
		}
	}
	return rankedPeer{}, false
}

func (c *Choice) hasChosenKeyOf(peer yggdrasilpeerlist.Peer) bool {
	for _, chosen := range c.chosenPeers {
		if chosen.peer.Key.Equal(peer.Key) {
			return true
		}
	}
	return false
}

func (c *Choice) mayReadList() bool {
	return c.listReadAt.IsZero() || c.clock.Now().Sub(c.listReadAt) >= listReadPause
}

func (c *Choice) readList(ctx context.Context) {
	c.listReadAt = c.clock.Now()
	c.fastPeers = fastestFirst(ctx, c.connectTimer, c.list.UpPeers(ctx))
	if len(c.fastPeers) == 0 {
		slog.WarnContext(ctx, "no public Yggdrasil peer connects")
	}
}

func (c *Choice) choose(ctx context.Context, next rankedPeer) {
	c.node.AddPeer(ctx, next.peer.URL)
	c.chosenPeers = append(c.chosenPeers, chosenPeer{peer: next.peer})
	slog.DebugContext(ctx, "public Yggdrasil peer chosen",
		slog.String("peer", next.peer.URL.Redacted()),
		slog.Duration("connectTime", next.connectTime),
	)
}
