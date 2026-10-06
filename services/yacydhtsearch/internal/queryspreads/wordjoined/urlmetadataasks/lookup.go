package urlmetadataasks

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
)

type Lookup struct {
	peerAsks              PeerAsks
	askLimits             askLimits
	cutoff                Cutoff
	clock                 Clock
	asksContext           context.Context
	cancelAsks            context.CancelFunc
	run                   *run
	outcomes              chan peerasks.URLMetadataAskOutcome
	settledPuts           chan struct{}
	ended                 chan struct{}
	amountOfUnsettledPuts int
}

func (lookup *Lookup) AskFor(holders documentholders.Holders) {
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
	answers := lookup.answersOnceEnded()
	lookup.stopAsking()

	return answers
}

func (lookup *Lookup) answersOnceEnded() Answers {
	graceEnded := make(chan struct{})
	stopGrace := func() {}
	defer func() { stopGrace() }()
	graceStarted := false
	for {
		if lookup.amountOfUnsettledPuts == 0 {
			return lookup.run.endedBy(EndedByEveryAskSettled)
		}
		if lookup.run.covered() {
			return lookup.run.endedBy(EndedByCoverage)
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
			return lookup.run.cutOff()
		}
	}
}

func (lookup *Lookup) stopAsking() {
	lookup.cancelAsks()
	close(lookup.ended)
}
