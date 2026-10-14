package distributionloop_test

import (
	"sync"
	"time"
)

type fakeObserver struct {
	guard                  sync.Mutex
	undeliveredPostings    int
	longestRefreshLateness time.Duration
	staleHoldersDropped    int
	postingsHandedOff      int
	batchesSkipped         map[string]int
	batchesCompleted       int
	batchesAborted         map[string]int
	holderRingFractions    []float64
	peersExcluded          int
	wordPartitionsWaiting  map[string]int
	peersPerSector         []int
}

func newFakeObserver() *fakeObserver {
	return &fakeObserver{
		batchesSkipped:        make(map[string]int),
		batchesAborted:        make(map[string]int),
		wordPartitionsWaiting: make(map[string]int),
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

func (f *fakeObserver) ObserveStaleHoldersDropped(amountOfStaleHolders int) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.staleHoldersDropped += amountOfStaleHolders
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

func (f *fakeObserver) ObservePeersAcceptingRemoteIndexPerDHTRingSector(peersPerSector []int) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.peersPerSector = peersPerSector
}

func (f *fakeObserver) ObserveWordPartitionsWaiting(
	waitReason string,
	amountOfWordPartitions int,
) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.wordPartitionsWaiting[waitReason] = amountOfWordPartitions
}

func (f *fakeObserver) wordPartitionsWaitingFor(waitReason string) int {
	f.guard.Lock()
	defer f.guard.Unlock()

	return f.wordPartitionsWaiting[waitReason]
}

func (f *fakeObserver) ObserveHolderRingFractions(ringFractions []float64) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.holderRingFractions = append(f.holderRingFractions, ringFractions...)
}

func (f *fakeObserver) ObservePeerExcluded() {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.peersExcluded++
}
