package distributionloop

type DHTRingObserver interface {
	ObserveReplicaRingFractions(ringFractions []float64)
}
