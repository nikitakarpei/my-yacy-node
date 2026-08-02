package rwidistribution

import "time"

type fakePostingOfferCycleObserver struct {
	offers                map[string]int
	postingsOffered       map[string]int
	urlMetadataDeliveries map[string]int
	urlsDelivered         map[string]int
	prunes                int
	gone                  int
	oldestDueAge          time.Duration
	cyclesSkipped         int
	replicationUnread     int
}

func newFakePostingOfferCycleObserver() *fakePostingOfferCycleObserver {
	return &fakePostingOfferCycleObserver{
		offers:                make(map[string]int),
		postingsOffered:       make(map[string]int),
		urlMetadataDeliveries: make(map[string]int),
		urlsDelivered:         make(map[string]int),
	}
}

func (f *fakePostingOfferCycleObserver) ObservePostingOffer(outcome string, postings int) {
	f.offers[outcome]++
	f.postingsOffered[outcome] += postings
}

func (f *fakePostingOfferCycleObserver) ObserveURLMetadataDelivery(outcome string, urls int) {
	f.urlMetadataDeliveries[outcome]++
	f.urlsDelivered[outcome] += urls
}

func (f *fakePostingOfferCycleObserver) ObservePostingsGone(gone int) {
	f.gone = gone
}

func (f *fakePostingOfferCycleObserver) ObserveOldestDuePostingAge(age time.Duration) {
	f.oldestDueAge = age
}

func (f *fakePostingOfferCycleObserver) ObserveLedgerPrune(dropped int) {
	f.prunes += dropped
}

func (f *fakePostingOfferCycleObserver) ObserveCycleSkipped() {
	f.cyclesSkipped++
}

func (f *fakePostingOfferCycleObserver) ObserveReplicationUnread() {
	f.replicationUnread++
}
