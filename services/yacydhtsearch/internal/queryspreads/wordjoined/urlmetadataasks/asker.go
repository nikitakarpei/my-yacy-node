// Package urlmetadataasks looks up the URL metadata of the joined documents of
// one query. Its lookup asks the peers that listed the documents as soon as they
// join. From End on, the lookup ends once the answers cover the documents, every
// ask settled, or the cutoff passed.
package urlmetadataasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type PeerAsks interface {
	AskForURLMetadata(
		ctx context.Context,
		asks []peerasks.URLMetadataAsk,
	) <-chan peerasks.URLMetadataAskOutcome
}

type Ceilings interface {
	CeilingOf(ctx context.Context, address string) int
}

type Clock interface {
	After(timeout time.Duration, expire func()) (stop func())
}

type Asker struct {
	peerAsks  PeerAsks
	askLimits askLimits
	cutoff    Cutoff
	clock     Clock
}

func New(
	peerAsks PeerAsks,
	ceilings Ceilings,
	cutoff Cutoff,
	clock Clock,
	networkRedundancy int,
) Asker {
	return Asker{
		peerAsks:  peerAsks,
		askLimits: askLimits{ceilings: ceilings, networkRedundancy: networkRedundancy},
		cutoff:    cutoff,
		clock:     clock,
	}
}

func (asker Asker) Begin(ctx context.Context) *Lookup {
	asksContext, cancelAsks := context.WithCancel(ctx)

	return &Lookup{
		peerAsks:    asker.peerAsks,
		askLimits:   asker.askLimits,
		cutoff:      asker.cutoff,
		clock:       asker.clock,
		asksContext: asksContext,
		cancelAsks:  cancelAsks,
		run:         noAsksPutYet(),
		outcomes:    make(chan peerasks.URLMetadataAskOutcome),
		settledPuts: make(chan struct{}),
		ended:       make(chan struct{}),
	}
}
