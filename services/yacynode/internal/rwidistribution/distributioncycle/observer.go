package distributioncycle

import "time"

// Observer reports what a posting offer cycle does, so an operator can tell
// whether stored postings are reaching their responsible peers.
type Observer interface {
	ObservePostingOffer(outcome string, postings int)
	ObserveURLMetadataDelivery(outcome string, urls int)
	ObservePostingsGone(gone int)
	ObserveScheduledPostings(postings int)
	ObserveLongestOfferLateness(lateness time.Duration)
	ObserveStaleReplicasDropped(dropped int)
	ObserveCycleSkipped(reason string)
}

type NoObserver struct{}

func (NoObserver) ObservePostingOffer(string, int) {}

func (NoObserver) ObserveURLMetadataDelivery(string, int) {}

func (NoObserver) ObservePostingsGone(int) {}

func (NoObserver) ObserveScheduledPostings(int) {}

func (NoObserver) ObserveLongestOfferLateness(time.Duration) {}

func (NoObserver) ObserveStaleReplicasDropped(int) {}

func (NoObserver) ObserveCycleSkipped(string) {}
