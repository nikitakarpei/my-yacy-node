// Package urlmetadataasks looks up the URL metadata of the joined documents of
// one query. Its lookup asks the peers that listed the documents as soon as they
// join and hands each answer to its recipient as it arrives. From End on, it ends
// once the answers cover the documents, every ask settled, or the cutoff passed,
// and reports how it performed to its observer.
package urlmetadataasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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

type Recipient interface {
	PeerSentURLMetadata(peer yacymodel.Hash, metadataOfEachDocument []yacymodel.URLMetadata)
}

type Clock interface {
	After(timeout time.Duration, expire func()) (stop func())
}

type Asker struct {
	peerAsks  PeerAsks
	askLimits askLimits
	cutoff    Cutoff
	clock     Clock
	observer  URLMetadataLookupObserver
}

//nolint:revive // argument-limit: the asker takes each port it asks through and each setting
func New(
	peerAsks PeerAsks,
	ceilings Ceilings,
	cutoff Cutoff,
	clock Clock,
	asksPerDocument int,
	observer URLMetadataLookupObserver,
) Asker {
	return Asker{
		peerAsks:  peerAsks,
		askLimits: askLimits{ceilings: ceilings, asksPerDocument: asksPerDocument},
		cutoff:    cutoff,
		clock:     clock,
		observer:  observer,
	}
}

func (asker Asker) Begin(ctx context.Context, recipient Recipient) *Lookup {
	asksContext, cancelAsks := context.WithCancel(ctx)
	lookup := &Lookup{
		ctx:          ctx,
		peerAsks:     asker.peerAsks,
		askLimits:    asker.askLimits,
		cutoff:       asker.cutoff,
		clock:        asker.clock,
		observer:     asker.observer,
		recipient:    recipient,
		asksContext:  asksContext,
		cancelAsks:   cancelAsks,
		run:          noAsksPutYet(),
		holdersToAsk: make(chan documentholders.Holders),
		holdersAsked: make(chan struct{}),
		outcomes:     make(chan peerasks.URLMetadataAskOutcome),
		settledPuts:  make(chan struct{}),
		endAsked:     make(chan struct{}),
		ended:        make(chan struct{}),
	}
	go lookup.settleUntilEnded()

	return lookup
}
