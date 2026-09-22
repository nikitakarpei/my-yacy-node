package distributioncycle_test

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
)

const cycleEndBuffer = 16

const shortfall = string(postingofferschedule.OfferOrderShortfall)

type fakeObserver struct {
	offers                map[string]int
	postingsOffered       map[string]int
	urlMetadataDeliveries map[string]int
	urlsDelivered         map[string]int
	urlsUnknownToUs       int
	staleReplicasDropped  int
	postingsHandedOff     int
	gone                  int
	scheduledPostings     map[string]int
	longestOfferLateness  map[string]time.Duration
	cyclesSkipped         map[string]int
	cyclesCompleted       int
	batchesAborted        map[string]int
	replicaRingFractions  []float64
	cycleEnds             chan struct{}
}

func newFakeObserver() *fakeObserver {
	return &fakeObserver{
		offers:                make(map[string]int),
		postingsOffered:       make(map[string]int),
		urlMetadataDeliveries: make(map[string]int),
		urlsDelivered:         make(map[string]int),
		cyclesSkipped:         make(map[string]int),
		batchesAborted:        make(map[string]int),
		scheduledPostings:     make(map[string]int),
		longestOfferLateness:  make(map[string]time.Duration),
		cycleEnds:             make(chan struct{}, cycleEndBuffer),
	}
}

func (f *fakeObserver) ObservePostingOffer(outcome string, postings int) {
	f.offers[outcome]++
	f.postingsOffered[outcome] += postings
}

func (f *fakeObserver) ObserveURLMetadataDelivery(outcome string, urls int) {
	f.urlMetadataDeliveries[outcome]++
	f.urlsDelivered[outcome] += urls
}

func (f *fakeObserver) ObserveURLsUnknownToUs(urls int) {
	f.urlsUnknownToUs += urls
}

func (f *fakeObserver) ObservePostingsGone(gone int) {
	f.gone = gone
}

func (f *fakeObserver) ObserveScheduledPostings(order string, postings int) {
	f.scheduledPostings[order] = postings
}

func (f *fakeObserver) ObserveLongestOfferLateness(order string, lateness time.Duration) {
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
	f.staleReplicasDropped += dropped
}

func (f *fakeObserver) ObservePostingsHandedOff(handedOff int) {
	f.postingsHandedOff += handedOff
}

func (f *fakeObserver) ObserveCycleSkipped(reason string) {
	f.cyclesSkipped[reason]++
	f.cycleEnds <- struct{}{}
}

func (f *fakeObserver) ObserveCycleCompleted() {
	f.cyclesCompleted++
	f.cycleEnds <- struct{}{}
}

func (f *fakeObserver) ObserveBatchAborted(reason string) {
	f.batchesAborted[reason]++
}

func (f *fakeObserver) ObservePeersAcceptingRemoteIndexPerDHTRingSector([]int) {}

func (f *fakeObserver) ObserveReplicaRingFractions(ringFractions []float64) {
	f.replicaRingFractions = append(f.replicaRingFractions, ringFractions...)
}
