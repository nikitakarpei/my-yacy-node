package distributionloop

type DHTRingObserver interface {
	ObserveHolderRingFractions(ringFractions []float64)
	ObservePeersAcceptingRemoteIndexPerDHTRingSector(peersPerSector []int)
}
