package rwidistribution

type fakePostingOfferCycleObserver struct {
	offers          map[string]int
	postingsOffered map[string]int
	prunes          int
	considered      int
	cyclesSkipped   int
}

func newFakePostingOfferCycleObserver() *fakePostingOfferCycleObserver {
	return &fakePostingOfferCycleObserver{
		offers:          make(map[string]int),
		postingsOffered: make(map[string]int),
	}
}

func (f *fakePostingOfferCycleObserver) ObservePostingOffer(outcome string, postings int) {
	f.offers[outcome]++
	f.postingsOffered[outcome] += postings
}

func (f *fakePostingOfferCycleObserver) ObservePostingsConsidered(considered int) {
	f.considered = considered
}

func (f *fakePostingOfferCycleObserver) ObserveLedgerPrune(dropped int) {
	f.prunes += dropped
}

func (f *fakePostingOfferCycleObserver) ObserveCycleSkipped() {
	f.cyclesSkipped++
}
