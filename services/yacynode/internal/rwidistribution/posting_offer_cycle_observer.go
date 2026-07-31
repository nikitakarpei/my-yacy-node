package rwidistribution

const (
	resultUnreachable = "unreachable"
	resultError       = "error"
)

// PostingOfferCycleObserver reports what a posting offer cycle does, so an
// operator can tell whether stored postings are reaching their responsible peers.
type PostingOfferCycleObserver interface {
	ObservePostingOffer(result string, postings int)
	ObserveScheduleDrain(drained int)
	ObserveLedgerPrune(dropped int)
	ObserveCycleSkipped(reachablePeers int)
}

type noPostingOfferCycleObserver struct{}

func (noPostingOfferCycleObserver) ObservePostingOffer(string, int) {}

func (noPostingOfferCycleObserver) ObserveScheduleDrain(int) {}

func (noPostingOfferCycleObserver) ObserveLedgerPrune(int) {}

func (noPostingOfferCycleObserver) ObserveCycleSkipped(int) {}
