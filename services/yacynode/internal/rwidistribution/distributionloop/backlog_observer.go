package distributionloop

type BacklogObserver interface {
	ObserveUndeliveredPostings(amountOfPostings int)
	ObserveArcs(amountPerState map[string]int)
}
