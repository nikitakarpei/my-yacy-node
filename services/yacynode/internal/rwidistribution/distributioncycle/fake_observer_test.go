package distributioncycle_test

import (
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
)

const cycleEndBuffer = 16

const shortfall = string(postingofferschedule.OfferOrderShortfall)

type fakeObserver struct {
	guard                sync.Mutex
	staleReplicasDropped int
	postingsHandedOff    int
	scheduledPostings    map[string]int
	longestOfferLateness map[string]time.Duration
	cyclesSkipped        map[string]int
	cyclesCompleted      int
	batchesAborted       map[string]int
	replicaRingFractions []float64
	cycleEnds            chan struct{}
}

func newFakeObserver() *fakeObserver {
	return &fakeObserver{
		cyclesSkipped:        make(map[string]int),
		batchesAborted:       make(map[string]int),
		scheduledPostings:    make(map[string]int),
		longestOfferLateness: make(map[string]time.Duration),
		cycleEnds:            make(chan struct{}, cycleEndBuffer),
	}
}

func (f *fakeObserver) ObserveScheduledPostings(order string, postings int) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.scheduledPostings[order] = postings
}

func (f *fakeObserver) ObserveLongestOfferLateness(order string, lateness time.Duration) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.longestOfferLateness[order] = lateness
}

func (f *fakeObserver) sumOfScheduledPostings() int {
	var sum int
	for _, postings := range f.scheduledPostings {
		sum += postings
	}

	return sum
}

func (f *fakeObserver) ObserveStaleReplicasDropped(dropped int) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.staleReplicasDropped += dropped
}

func (f *fakeObserver) ObservePostingHandedOff() {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.postingsHandedOff++
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
