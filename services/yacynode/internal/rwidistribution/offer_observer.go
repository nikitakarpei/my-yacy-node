package rwidistribution

const (
	offerResultUnreachable = "unreachable"
	offerResultError       = "error"
)

// OfferObserver reports what a distribution cycle does, so an operator can
// tell whether stored postings are reaching their responsible peers.
type OfferObserver interface {
	ObserveOffer(result string, postings int)
	ObserveScheduleDrain(drained int)
	ObserveLedgerPrune(dropped int)
	ObserveCycleSkipped(reachablePeers int)
}

type noOfferObserver struct{}

func (noOfferObserver) ObserveOffer(string, int) {}

func (noOfferObserver) ObserveScheduleDrain(int) {}

func (noOfferObserver) ObserveLedgerPrune(int) {}

func (noOfferObserver) ObserveCycleSkipped(int) {}
