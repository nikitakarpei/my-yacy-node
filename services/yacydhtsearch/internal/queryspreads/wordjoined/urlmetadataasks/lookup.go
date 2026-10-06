package urlmetadataasks

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
)

type Lookup struct {
	ctx                   context.Context
	peerAsks              PeerAsks
	askLimits             askLimits
	cutoff                Cutoff
	clock                 Clock
	observer              URLMetadataLookupObserver
	asksContext           context.Context
	cancelAsks            context.CancelFunc
	run                   *run
	outcomes              chan peerasks.URLMetadataAskOutcome
	settledPuts           chan struct{}
	ended                 chan struct{}
	amountOfUnsettledPuts int
}

func (lookup *Lookup) AskFor(holders documentholders.Holders) {
	lookup.run.give(holders.MostHeldFirst())
	asks := lookup.askLimits.asksFor(lookup.asksContext, holders)
	if len(asks) == 0 {
		return
	}
	lookup.run.add(asks)
	lookup.put(asks)
}

func (lookup *Lookup) put(asks []peerasks.URLMetadataAsk) {
	lookup.amountOfUnsettledPuts++
	go lookup.forward(lookup.peerAsks.AskForURLMetadata(lookup.asksContext, asks))
}

func (lookup *Lookup) forward(outcomesOfThePut <-chan peerasks.URLMetadataAskOutcome) {
	for outcome := range outcomesOfThePut {
		select {
		case lookup.outcomes <- outcome:
		case <-lookup.ended:
			return
		}
	}
	select {
	case lookup.settledPuts <- struct{}{}:
	case <-lookup.ended:
	}
}

func (lookup *Lookup) End() Answers {
	lookup.waitUntilEnded()
	lookup.stopAsking()
	lookup.observer.URLMetadataLookupPerformed(lookup.ctx, lookup.run.performed())

	return lookup.run.answers()
}

func (lookup *Lookup) waitUntilEnded() {
	graceEnded := make(chan struct{})
	stopGrace := func() {}
	defer func() { stopGrace() }()
	graceStarted := false
	for {
		if lookup.amountOfUnsettledPuts == 0 {
			lookup.run.endBy(EndedByEveryAskSettled)

			return
		}
		if lookup.run.covered() {
			lookup.run.endBy(EndedByCoverage)

			return
		}
		select {
		case outcome := <-lookup.outcomes:
			lookup.run.settle(outcome)
			if !graceStarted && lookup.cutoff.reachedBy(lookup.run.settledShare()) {
				graceStarted = true
				stopGrace = lookup.clock.After(lookup.cutoff.Grace, func() { close(graceEnded) })
			}
		case <-lookup.settledPuts:
			lookup.amountOfUnsettledPuts--
		case <-graceEnded:
			lookup.run.cutOff()

			return
		}
	}
}

func (lookup *Lookup) stopAsking() {
	lookup.cancelAsks()
	close(lookup.ended)
}
