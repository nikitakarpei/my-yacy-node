package distributionloop

import "time"

type BacklogObserver interface {
	ObserveUndeliveredPostings(amountOfPostings int)
	ObserveLongestRefreshLateness(lateness time.Duration)
	ObserveArcs(amountPerState map[string]int)
}
