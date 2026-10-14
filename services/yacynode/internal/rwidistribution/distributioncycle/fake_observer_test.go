package distributioncycle_test

import (
	"sync"
	"time"
)

const cycleEndBuffer = 16

type fakeObserver struct {
	guard                  sync.Mutex
	undeliveredPostings    int
	longestRefreshLateness time.Duration
	staleReplicasDropped   int
	postingsHandedOff      int
	cyclesSkipped          map[string]int
	cyclesCompleted        int
	batchesAborted         map[string]int
	replicaRingFractions   []float64
	cycleEnds              chan struct{}
}

func newFakeObserver() *fakeObserver {
	return &fakeObserver{
		cyclesSkipped:  make(map[string]int),
		batchesAborted: make(map[string]int),
		cycleEnds:      make(chan struct{}, cycleEndBuffer),
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

func (f *fakeObserver) ObserveCycleSkipped(reason string) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.cyclesSkipped[reason]++
	f.cycleEnds <- struct{}{}
}

func (f *fakeObserver) ObserveCycleCompleted() {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.cyclesCompleted++
	f.cycleEnds <- struct{}{}
}

func (f *fakeObserver) ObserveBatchAborted(reason string) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.batchesAborted[reason]++
}

func (f *fakeObserver) ObservePeersAcceptingRemoteIndexPerDHTRingSector([]int) {}

func (f *fakeObserver) ObserveReplicaRingFractions(ringFractions []float64) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.replicaRingFractions = append(f.replicaRingFractions, ringFractions...)
}
