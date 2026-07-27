package peerroster

type RosterObserver interface {
	ObserveKnownPeers(count int)
	ObserveReachablePeers(count int)
}

type discardObserver struct{}

func (discardObserver) ObserveKnownPeers(int) {}

func (discardObserver) ObserveReachablePeers(int) {}

var DiscardObserver RosterObserver = discardObserver{}
