package distributionloop

type BatchObserver interface {
	ObserveStaleReplicasDropped(droppedReplicas int)
	ObservePostingsHandedOff(amountOfPostings int)
	ObserveBatchSkipped(reason string)
	ObserveBatchCompleted()
	ObserveBatchAborted(reason string)
}
