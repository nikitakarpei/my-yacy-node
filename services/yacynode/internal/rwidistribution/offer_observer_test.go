package rwidistribution

type fakeOfferObserver struct {
	offers          map[string]int
	postingsOffered map[string]int
	prunes          int
	drained         int
	cyclesSkipped   int
}

func newFakeOfferObserver() *fakeOfferObserver {
	return &fakeOfferObserver{offers: make(map[string]int), postingsOffered: make(map[string]int)}
}

func (f *fakeOfferObserver) ObserveOffer(result string, postings int) {
	f.offers[result]++
	f.postingsOffered[result] += postings
}

func (f *fakeOfferObserver) ObserveScheduleDrain(drained int) {
	f.drained = drained
}

func (f *fakeOfferObserver) ObserveLedgerPrune(dropped int) {
	f.prunes += dropped
}

func (f *fakeOfferObserver) ObserveCycleSkipped(int) {
	f.cyclesSkipped++
}
