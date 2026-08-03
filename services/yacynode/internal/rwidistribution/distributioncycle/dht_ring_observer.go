package distributioncycle

type DHTRingObserver interface {
	ObserveReplicaRingFractions(ringFractions []float64)
}
