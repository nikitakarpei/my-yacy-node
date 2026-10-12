package distributionloop

type DHTRingObserver interface {
	ObserveReplicaRingFractions(ringFractions []float64)
	ObservePeersAcceptingRemoteIndexPerDHTRingSector(peersPerSector []int)
}
