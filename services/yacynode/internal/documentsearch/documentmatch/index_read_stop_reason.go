package documentmatch

type IndexReadStopReason int

const (
	IndexReadStoppedAtEndOfTerm IndexReadStopReason = iota
	IndexReadStoppedAtRelevanceBound
	IndexReadStoppedAtDeadline
)
