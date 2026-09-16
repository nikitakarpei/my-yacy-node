// Package peerdirectoryrefresh keeps the peer directory current: it re-reads
// the seedlists and probes which address of each known peer answers. It probes
// the address a peer has earned the most presence at first, so an address a
// seedlist has only just named for that peer answers for it only when every
// address this deployment has already heard from is silent.
package peerdirectoryrefresh

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenesswire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacyseedlist"
)

type PeerPresence interface {
	EarnedPresenceOf(
		ctx context.Context,
		peerAtAddress peeranswerhistory.PeerAtAddress,
	) time.Duration
}

type ProbeLimits struct {
	ProbeBudget    time.Duration
	ProbesInFlight int
}

type Cycle struct {
	seedlists yacyseedlist.Seedlists
	directory *peerdirectory.Directory
	liveness  peerlivenesswire.Wire
	presence  PeerPresence
	probes    ProbeLimits
}

func New(
	seedlists yacyseedlist.Seedlists,
	directory *peerdirectory.Directory,
	liveness peerlivenesswire.Wire,
	presence PeerPresence,
	probes ProbeLimits,
) Cycle {
	return Cycle{
		seedlists: seedlists,
		directory: directory,
		liveness:  liveness,
		presence:  presence,
		probes:    probes,
	}
}

func (c Cycle) Run(ctx context.Context, every time.Duration) {
	ticks := time.NewTicker(every)
	defer ticks.Stop()

	for {
		c.RefreshOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticks.C:
		}
	}
}

func (c Cycle) RefreshOnce(ctx context.Context) {
	c.directory.Admit(ctx, c.seedlists.Fetch(ctx))
	c.probeKnownPeers(ctx, c.directory.KnownPeers(ctx))
}

func (c Cycle) probeKnownPeers(ctx context.Context, knownPeers []peerdirectory.KnownPeer) {
	inFlight := make(chan struct{}, c.probes.ProbesInFlight)
	var probes sync.WaitGroup

	for _, peer := range knownPeers {
		probes.Add(1)
		go func() {
			defer probes.Done()
			inFlight <- struct{}{}
			defer func() { <-inFlight }()
			c.probeOne(ctx, peer)
		}()
	}
	probes.Wait()
}

func (c Cycle) probeOne(ctx context.Context, peer peerdirectory.KnownPeer) {
	for _, address := range c.addressesMostPresentFirst(ctx, peer) {
		probeCtx, endProbe := context.WithTimeout(ctx, c.probes.ProbeBudget)
		alive := c.liveness.Alive(probeCtx, address)
		endProbe()
		if alive {
			c.directory.ConfirmAnswering(ctx, peer.Hash, address)

			return
		}
	}
	c.directory.ConfirmSilent(ctx, peer.Hash)
}

func (c Cycle) addressesMostPresentFirst(
	ctx context.Context,
	peer peerdirectory.KnownPeer,
) []string {
	mostPresentFirst := slices.Clone(peer.Addresses)
	slices.SortStableFunc(mostPresentFirst, func(a, b string) int {
		return cmp.Compare(
			c.presence.EarnedPresenceOf(
				ctx, peeranswerhistory.PeerAtAddress{Hash: peer.Hash, Address: b},
			),
			c.presence.EarnedPresenceOf(
				ctx, peeranswerhistory.PeerAtAddress{Hash: peer.Hash, Address: a},
			),
		)
	})

	return mostPresentFirst
}
