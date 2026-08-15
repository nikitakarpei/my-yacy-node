package distributioncycle

type CycleObserver interface {
	ObservePostingsGone(gonePostings int)
	ObserveStaleReplicasDropped(droppedReplicas int)
	ObservePostingsHandedOff(handedOffPostings int)
	ObserveCycleSkipped(reason string)
	ObserveCycleCompleted()
	ObserveBatchAborted(reason string)
}
