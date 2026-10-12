package distributionloop_test

import (
	"sync"
	"time"
)

type fakeObserver struct {
	guard                  sync.Mutex
	undeliveredPostings    int
	longestRefreshLateness time.Duration
	staleReplicasDropped   int
	postingsHandedOff      int
	batchesSkipped         map[string]int
	batchesCompleted       int
	batchesAborted         map[string]int
	replicaRingFractions   []float64
	peersExcluded          int
	wordPartitionsWaiting  map[skipKind]int
}

type skipKind struct {
	offerKind  string
	skipReason string
}

func newFakeObserver() *fakeObserver {
	return &fakeObserver{
		batchesSkipped:        make(map[string]int),
		batchesAborted:        make(map[string]int),
		wordPartitionsWaiting: make(map[skipKind]int),
	}
}

func (f *fakeObserver) ObserveUndeliveredPostings(amountOfPostings int) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.undeliveredPostings = amountOfPostings
}

func (f *fakeObserver) ObserveLongestRefreshLateness(lateness time.Duration) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.longestRefreshLateness = lateness
}

func (f *fakeObserver) ObserveStaleReplicasDropped(dropped int) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.staleReplicasDropped += dropped
}

func (f *fakeObserver) ObservePostingsHandedOff(amountOfPostings int) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.postingsHandedOff += amountOfPostings
}

func (f *fakeObserver) ObserveBatchSkipped(reason string) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.batchesSkipped[reason]++
}

func (f *fakeObserver) ObserveBatchCompleted() {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.batchesCompleted++
}

func (f *fakeObserver) ObserveBatchAborted(reason string) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.batchesAborted[reason]++
}

func (f *fakeObserver) ObservePeersAcceptingRemoteIndexPerDHTRingSector([]int) {}

func (f *fakeObserver) ObserveWordPartitionsWaiting(
	offerKind string,
	skipReason string,
	amountOfWordPartitions int,
) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.wordPartitionsWaiting[skipKind{offerKind: offerKind, skipReason: skipReason}] = amountOfWordPartitions
}

func (f *fakeObserver) ObserveReplicaRingFractions(ringFractions []float64) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.replicaRingFractions = append(f.replicaRingFractions, ringFractions...)
}

func (f *fakeObserver) ObservePeerExcluded() {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.peersExcluded++
}
