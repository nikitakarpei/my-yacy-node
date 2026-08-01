package rwidistribution

import "time"

// PostingOfferCycleObserver reports what a posting offer cycle does, so an
// operator can tell whether stored postings are reaching their responsible peers.
type PostingOfferCycleObserver interface {
	ObservePostingOffer(outcome string, postings int)
	ObservePostingsDue(due int)
	ObservePostingsGone(gone int)
	ObserveOldestDuePostingAge(age time.Duration)
	ObserveLedgerPrune(dropped int)
	ObserveCycleSkipped()
}

type noPostingOfferCycleObserver struct{}

func (noPostingOfferCycleObserver) ObservePostingOffer(string, int) {}

func (noPostingOfferCycleObserver) ObservePostingsDue(int) {}

func (noPostingOfferCycleObserver) ObservePostingsGone(int) {}

func (noPostingOfferCycleObserver) ObserveOldestDuePostingAge(time.Duration) {}

func (noPostingOfferCycleObserver) ObserveLedgerPrune(int) {}

func (noPostingOfferCycleObserver) ObserveCycleSkipped() {}
