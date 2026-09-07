// Package peerdirectoryrefresh keeps the peer directory current: it re-reads
// the seedlists and probes which address of each known peer answers.
package peerdirectoryrefresh

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenesswire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacyseedlist"
)

type Cycle struct {
	seedlists   yacyseedlist.Seedlists
	directory   *peerdirectory.Directory
	liveness    peerlivenesswire.Wire
	interval    time.Duration
	probeBudget time.Duration
	inFlight    int
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators
func New(
	seedlists yacyseedlist.Seedlists,
	directory *peerdirectory.Directory,
	liveness peerlivenesswire.Wire,
	interval, probeBudget time.Duration,
	inFlight int,
) Cycle {
	return Cycle{
		seedlists:   seedlists,
		directory:   directory,
		liveness:    liveness,
		interval:    interval,
		probeBudget: probeBudget,
		inFlight:    inFlight,
	}
}

func (c Cycle) Run(ctx context.Context) {
	ticks := time.NewTicker(c.interval)
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
	c.probeKnownPeers(ctx)
}

func (c Cycle) probeKnownPeers(ctx context.Context) {
	inFlight := make(chan struct{}, c.inFlight)
	var probes sync.WaitGroup

	for _, peer := range c.directory.KnownPeers(ctx) {
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
	for _, address := range peer.Addresses {
		probeCtx, endProbe := context.WithTimeout(ctx, c.probeBudget)
		alive := c.liveness.Alive(probeCtx, address)
		endProbe()
		if alive {
			c.directory.ConfirmAnswering(ctx, peer.Hash, address)
			return
		}
	}
	c.directory.ConfirmSilent(ctx, peer.Hash)
}
