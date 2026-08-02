package distributioncycle

type Observer interface {
	ObservePostingsGone(gonePostings int)
	ObserveStaleReplicasDropped(droppedReplicas int)
	ObservePostingsHandedOff(handedOffPostings int)
	ObserveCycleSkipped(reason string)
}
