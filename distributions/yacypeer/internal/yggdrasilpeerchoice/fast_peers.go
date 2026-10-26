package yggdrasilpeerchoice

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilpeerlist"
)

const parallelConnects = 16

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
	connectTimes := make([]time.Duration, len(peers))
	connected := make([]bool, len(peers))
	slots := make(chan struct{}, parallelConnects)
	var connects sync.WaitGroup
	for index, peer := range peers {
		slots <- struct{}{}
		connects.Go(func() {
			defer func() { <-slots }()
			connectTimes[index], connected[index] = connectTimer.ConnectTimeOf(ctx, peer.URL.Host)
		})
	}
	connects.Wait()
	connectedPeers := make([]rankedPeer, 0, len(peers))
	for index, peer := range peers {
		if connected[index] {
			connectedPeers = append(connectedPeers,
				rankedPeer{peer: peer, connectTime: connectTimes[index]})
		}
	}
	return connectedPeers
}
