// Package yggdrasilpeerchoice chooses the public peers of the Yggdrasil node. It peers the node
// with the public peers that the peer list states are up and that connect fastest from here,
// each with its own key. It chooses once, and keeps a chosen peer that goes down.
package yggdrasilpeerchoice

import (
	"context"
	"crypto/ed25519"
	"log/slog"
	"net/url"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilpeerlist"
)

const wantedPeers = 3

type PeerList interface {
	UpPeers(ctx context.Context) []yggdrasilpeerlist.Peer
}

type Node interface {
	AddPeer(ctx context.Context, peerURL *url.URL, key ed25519.PublicKey)
}

type ConnectTimer interface {
	ConnectTimeOf(ctx context.Context, address string) (time.Duration, bool)
}

type Choice struct {
	list         PeerList
	node         Node
	connectTimer ConnectTimer
}

func New(list PeerList, node Node, connectTimer ConnectTimer) *Choice {
	return &Choice{list: list, node: node, connectTimer: connectTimer}
}

func (c *Choice) Choose(ctx context.Context) {
	fastPeers := fastestFirst(ctx, c.connectTimer, c.list.UpPeers(ctx))
	c.peerWith(ctx, fastestWithOwnKeysAmong(fastPeers))
}

func fastestWithOwnKeysAmong(fastPeers []rankedPeer) []rankedPeer {
	var chosenPeers []rankedPeer
	for _, fast := range fastPeers {
		if len(chosenPeers) == wantedPeers {
			break
		}
		if !slices.ContainsFunc(chosenPeers, func(chosen rankedPeer) bool {
			return chosen.peer.Key.Equal(fast.peer.Key)
		}) {
			chosenPeers = append(chosenPeers, fast)
		}
	}
	return chosenPeers
}

func (c *Choice) peerWith(ctx context.Context, chosenPeers []rankedPeer) {
	if len(chosenPeers) == 0 {
		slog.WarnContext(ctx, "no public Yggdrasil peer connects")
		return
	}
	for _, chosen := range chosenPeers {
		c.node.AddPeer(ctx, chosen.peer.URL, chosen.peer.Key)
		slog.DebugContext(ctx, "public Yggdrasil peer chosen",
			slog.String("peer", chosen.peer.URL.Redacted()),
			slog.Duration("connectTime", chosen.connectTime),
		)
	}
}
