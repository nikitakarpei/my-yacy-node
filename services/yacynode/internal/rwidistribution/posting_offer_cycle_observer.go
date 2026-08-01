package rwidistribution

// PostingOfferCycleObserver reports what a posting offer cycle does, so an
// operator can tell whether stored postings are reaching their responsible peers.
type PostingOfferCycleObserver interface {
	ObservePostingOffer(outcome string, postings int)
	ObservePostingsConsidered(considered int)
	ObserveLedgerPrune(dropped int)
	ObserveCycleSkipped()
}

type noPostingOfferCycleObserver struct{}

func (noPostingOfferCycleObserver) ObservePostingOffer(string, int) {}

func (noPostingOfferCycleObserver) ObservePostingsConsidered(int) {}

func (noPostingOfferCycleObserver) ObserveLedgerPrune(int) {}

func (noPostingOfferCycleObserver) ObserveCycleSkipped() {}
