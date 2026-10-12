package distributionloop

import (
	"maps"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

type waitingKind struct {
	offer  wordpartitionoffer.OfferKind
	reason wordpartitionoffer.SkipReason
}

type waitingWordPartitions struct {
	guard         sync.Mutex
	passesSoFar   map[yacymodel.DHTRingSector]map[waitingKind]int
	lastPasses    map[yacymodel.DHTRingSector]map[waitingKind]int
	reportedKinds map[waitingKind]struct{}
}

func newWaitingWordPartitions() *waitingWordPartitions {
	return &waitingWordPartitions{
		passesSoFar:   make(map[yacymodel.DHTRingSector]map[waitingKind]int),
		lastPasses:    make(map[yacymodel.DHTRingSector]map[waitingKind]int),
		reportedKinds: make(map[waitingKind]struct{}),
	}
}

func (w *waitingWordPartitions) add(
	sector yacymodel.DHTRingSector,
	skips []wordpartitionoffer.Skip,
) {
	w.guard.Lock()
	defer w.guard.Unlock()

	passSoFar, started := w.passesSoFar[sector]
	if !started {
		passSoFar = make(map[waitingKind]int)
		w.passesSoFar[sector] = passSoFar
	}
	for _, skip := range skips {
		passSoFar[waitingKind{offer: skip.Offer, reason: skip.Reason}]++
	}
}

func (w *waitingWordPartitions) endPassIn(sector yacymodel.DHTRingSector) map[waitingKind]int {
	w.guard.Lock()
	defer w.guard.Unlock()

	w.lastPasses[sector] = maps.Clone(w.passesSoFar[sector])
	delete(w.passesSoFar, sector)

	return w.amountsPerKind()
}

func (w *waitingWordPartitions) amountsPerKind() map[waitingKind]int {
	amounts := make(map[waitingKind]int, len(w.reportedKinds))
	for kind := range w.reportedKinds {
		amounts[kind] = 0
	}
	for _, lastPass := range w.lastPasses {
		for kind, amountOfWordPartitions := range lastPass {
			amounts[kind] += amountOfWordPartitions
			w.reportedKinds[kind] = struct{}{}
		}
	}

	return amounts
}
