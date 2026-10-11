package distributioncycle

type CycleObserver interface {
	ObservePostingsGone(gonePostings int)
	ObserveStaleReplicasDropped(droppedReplicas int)
	ObservePostingHandedOff()
	ObserveCycleSkipped(reason string)
	ObserveCycleCompleted()
	ObserveBatchAborted(reason string)
}
