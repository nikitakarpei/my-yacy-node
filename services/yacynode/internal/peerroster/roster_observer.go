package peerroster

type RosterObserver interface {
	ObserveKnownPeers(count int)
	ObserveReachablePeers(count int)
	ObserveDroppedSeed(reason SeedDropReason)
}

type discardObserver struct{}

func (discardObserver) ObserveKnownPeers(int) {}

func (discardObserver) ObserveReachablePeers(int) {}

func (discardObserver) ObserveDroppedSeed(SeedDropReason) {}

var DiscardObserver RosterObserver = discardObserver{}
