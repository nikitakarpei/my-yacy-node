package distributioncycle

import "time"

// Observer reports what a posting offer cycle does, so an operator can tell
// whether stored postings are reaching their responsible peers.
type Observer interface {
	ObservePostingOffer(outcome string, postings int)
	ObserveURLMetadataDelivery(outcome string, urls int)
	ObservePostingsGone(gone int)
	ObserveOldestDuePostingAge(age time.Duration)
	ObserveStaleReplicasDropped(dropped int)
	ObservePostingsAtLongestOfferWait(postings int)
	ObserveIneligibleReplicaRecipients(peers int)
	ObserveCycleSkipped()
	ObserveShortfallUnread()
}

type NoObserver struct{}

func (NoObserver) ObservePostingOffer(string, int) {}

func (NoObserver) ObserveURLMetadataDelivery(string, int) {}

func (NoObserver) ObservePostingsGone(int) {}

func (NoObserver) ObserveOldestDuePostingAge(time.Duration) {}

func (NoObserver) ObserveStaleReplicasDropped(int) {}

func (NoObserver) ObservePostingsAtLongestOfferWait(int) {}

func (NoObserver) ObserveIneligibleReplicaRecipients(int) {}

func (NoObserver) ObserveCycleSkipped() {}

func (NoObserver) ObserveShortfallUnread() {}
