package yggdrasilpeerchoice

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilpeerlist"
)

const parallelConnects = 64

type rankedPeer struct {
	peer        yggdrasilpeerlist.Peer
	connectTime time.Duration
}

func fastestFirst(
	ctx context.Context,
	connectTimer ConnectTimer,
	peers []yggdrasilpeerlist.Peer,
) []rankedPeer {
	rankedPeers := connectedPeersOf(ctx, connectTimer, peers)
	slices.SortStableFunc(rankedPeers, func(a, b rankedPeer) int {
		return cmp.Compare(a.connectTime, b.connectTime)
	})
	return rankedPeers
}

func connectedPeersOf(
	ctx context.Context,
	connectTimer ConnectTimer,
	peers []yggdrasilpeerlist.Peer,
) []rankedPeer {
	timedPeers := make([]rankedPeer, len(peers))
	slots := make(chan struct{}, parallelConnects)
	var connects sync.WaitGroup
	for index, peer := range peers {
		slots <- struct{}{}
		connects.Go(func() {
			defer func() { <-slots }()
			if connectTime, connected := connectTimer.ConnectTimeOf(ctx, peer.URL.Host); connected {
				timedPeers[index] = rankedPeer{peer: peer, connectTime: connectTime}
			}
		})
	}
	connects.Wait()
	return slices.DeleteFunc(
		timedPeers,
		func(timed rankedPeer) bool { return timed.peer.URL == nil },
	)
}
