package distributioncycle

import "time"

type fakeObserver struct {
	offers                map[string]int
	postingsOffered       map[string]int
	urlMetadataDeliveries map[string]int
	urlsDelivered         map[string]int
	staleReplicasDropped  int
	postingsAtLongestWait int
	ineligibleRecipients  int
	gone                  int
	oldestDueAge          time.Duration
	cyclesSkipped         int
	shortfallUnread       int
}

func newFakeObserver() *fakeObserver {
	return &fakeObserver{
		offers:                make(map[string]int),
		postingsOffered:       make(map[string]int),
		urlMetadataDeliveries: make(map[string]int),
		urlsDelivered:         make(map[string]int),
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

func (f *fakeObserver) ObservePostingsGone(gone int) {
	f.gone = gone
}

func (f *fakeObserver) ObserveOldestDuePostingAge(age time.Duration) {
	f.oldestDueAge = age
}

func (f *fakeObserver) ObserveStaleReplicasDropped(dropped int) {
	f.staleReplicasDropped += dropped
}

func (f *fakeObserver) ObservePostingsAtLongestOfferWait(postings int) {
	f.postingsAtLongestWait = postings
}

func (f *fakeObserver) ObserveIneligibleReplicaRecipients(peers int) {
	f.ineligibleRecipients = peers
}

func (f *fakeObserver) ObserveCycleSkipped() {
	f.cyclesSkipped++
}

func (f *fakeObserver) ObserveShortfallUnread() {
	f.shortfallUnread++
}
