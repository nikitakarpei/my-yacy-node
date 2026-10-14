package distributioncycle

import "time"

type CycleObserver interface {
	ObserveUndeliveredPostings(amountOfPostings int)
	ObserveLongestRefreshLateness(lateness time.Duration)
	ObserveStaleReplicasDropped(droppedReplicas int)
	ObservePostingsHandedOff(amountOfPostings int)
	ObserveCycleSkipped(reason string)
	ObserveCycleCompleted()
	ObserveBatchAborted(reason string)
}
