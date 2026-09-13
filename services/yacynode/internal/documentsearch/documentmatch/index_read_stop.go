package documentmatch

type IndexReadStop int

const (
	IndexReadStoppedAtEndOfTerm IndexReadStop = iota
	IndexReadStoppedAtRelevanceBound
	IndexReadStoppedAtDeadline
)
