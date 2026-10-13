package distributionloop

type BatchObserver interface {
	ObserveStaleHoldersDropped(amountOfStaleHolders int)
	ObservePostingsHandedOff(amountOfPostings int)
	ObservePostingsDelivered(amountOfPostings int)
	ObserveBatchCompleted()
	ObserveBatchAborted(reason string)
	ObservePostingsPerSend(amountOfPostings int)
}
