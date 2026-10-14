package distributionloop

import "time"

type BacklogObserver interface {
	ObserveUndeliveredPostings(amountOfPostings int)
	ObserveLongestDueWait(wait time.Duration)
	ObserveArcs(amountPerState map[string]int)
}
