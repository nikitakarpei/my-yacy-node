package distributionloop

type BatchObserver interface {
	ObserveStaleHoldersDropped(amountOfStaleHolders int)
	ObservePostingsHandedOff(amountOfPostings int)
	ObserveBatchSkipped(reason string)
	ObserveBatchCompleted()
	ObserveBatchAborted(reason string)
	ObserveWordPartitionsWaiting(waitReason string, amountOfWordPartitions int)
}
