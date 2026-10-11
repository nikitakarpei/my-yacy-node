package distributioncycle

type CycleObserver interface {
	ObserveStaleReplicasDropped(droppedReplicas int)
	ObservePostingHandedOff()
	ObserveCycleSkipped(reason string)
	ObserveCycleCompleted()
	ObserveBatchAborted(reason string)
}
