package distributionloop_test

import (
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributionloop"
)

type fakeObserver struct {
	guard               sync.Mutex
	undeliveredPostings int
	arcsPerState        map[string]int
	staleHoldersDropped int
	postingsHandedOff   int
	postingsDelivered   int
	batchesCompleted    int
	batchesAborted      map[string]int
	holderRingFractions []float64
	peersExcluded       int
	peersPerSector      []int
	postingsPerSend     []int
}

func newFakeObserver() *fakeObserver {
	return &fakeObserver{
		batchesAborted: make(map[string]int),
	}
}

func (f *fakeObserver) ObserveUndeliveredPostings(amountOfPostings int) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.undeliveredPostings = amountOfPostings
}

func (f *fakeObserver) ObserveArcs(amountPerState map[string]int) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.arcsPerState = amountPerState
}

func (f *fakeObserver) arcsIn(state distributionloop.ArcState) int {
	f.guard.Lock()
	defer f.guard.Unlock()

	return f.arcsPerState[string(state)]
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

func (f *fakeObserver) ObservePostingsDelivered(amountOfPostings int) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.postingsDelivered += amountOfPostings
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

func (f *fakeObserver) ObserveHolderRingFractions(ringFractions []float64) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.holderRingFractions = append(f.holderRingFractions, ringFractions...)
}

func (f *fakeObserver) ObservePeerExcluded(string) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.peersExcluded++
}

func (f *fakeObserver) ObservePostingsPerSend(amountOfPostings int) {
	f.guard.Lock()
	defer f.guard.Unlock()

	f.postingsPerSend = append(f.postingsPerSend, amountOfPostings)
}

func (f *fakeObserver) sortedPostingsPerSend() []int {
	f.guard.Lock()
	defer f.guard.Unlock()

	return slices.Sorted(slices.Values(f.postingsPerSend))
}

func (f *fakeObserver) ObservePeerSendTime(time.Duration) {}

func (f *fakeObserver) ObservePeersAtSendLimit(int) {}
